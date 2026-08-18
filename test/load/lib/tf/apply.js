// Create / poll / destroy helpers for tf-stress.

import http from 'k6/http';
import {check, sleep} from 'k6';

import {checkStatus} from '../checks.js';
import {authHeaders} from '../auth.js';
import {logIfUnexpected} from '../http.js';
import {recordResponse} from '../status.js';
import {
  blockStorageBody,
  instanceBody,
  networkBody,
  nicBody,
  routeTableBody,
  subnetBody,
  workspaceBody,
} from './bodies.js';
import {allPollTargets, netStoragePollTargets} from './plan.js';
import {tfUrls} from './urls.js';

// 404 is normal for poll-before-ready races and delete cleanup; 202 for DELETE.
// Without this, k6 counts them as http_req_failed and trips thresholds.
export function tfExpectStatuses() {
  http.setResponseCallback(http.expectedStatuses(200, 201, 202, 404));
}

/**
 * @param {import('../config.js').LoadConfig} cfg
 * @param {object} [extra]
 */
function params(cfg, extra) {
  return {
    headers: authHeaders(cfg),
    tags: (extra && extra.tags) || {},
  };
}

/**
 * @param {string} url
 * @param {object} body
 * @param {import('../config.js').LoadConfig} cfg
 * @param {string} label
 * @param {string} resource
 */
export function tfPut(url, body, cfg, label, resource) {
  const payload = JSON.stringify(body);
  const p = params(cfg, { tags: { name: 'tf_write', resource } });
  // Retry on admission throttle / gateway 5xx (common under parallel TF users).
  const maxAttempts = Number(__ENV.TF_WRITE_RETRIES || '8');
  let res;
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    res = http.put(url, payload, p);
    recordResponse(res, { name: 'tf_write', resource });
    if (res.status === 200 || res.status === 201) {
      break;
    }
    if (res.status === 429 || res.status === 500 || res.status === 503) {
      sleep(0.25 * attempt + Math.random() * 0.5);
      continue;
    }
    break;
  }
  logIfUnexpected(res, [200, 201], label);
  check(res, {
    [`${label} status is 200|201`]: (r) => r.status === 200 || r.status === 201,
  });
  return res;
}

/**
 * Parallel GETs for given targets.
 * @param {import('../config.js').LoadConfig} cfg
 * @param {{url: string, resource: string}[]} targets
 * @returns {number} batch size
 */
export function tfPollTargets(cfg, targets) {
  if (!targets.length) {
    return 0;
  }
  // Pad batch to at least 10 concurrent requests when few resources (duplicate GETs).
  const minBatch = Number(__ENV.TF_MIN_POLL_BATCH || '10');
  const expanded = targets.slice();
  while (expanded.length < minBatch) {
    expanded.push(targets[expanded.length % targets.length]);
  }
  const reqs = expanded.map((t) => [
    'GET',
    t.url,
    null,
    params(cfg, { tags: { name: 'tf_poll', resource: t.resource } }),
  ]);
  const responses = http.batch(reqs);
  let ok = 0;
  for (let i = 0; i < responses.length; i++) {
    const t = expanded[i];
    recordResponse(responses[i], {
      name: 'tf_poll',
      resource: t && t.resource,
    });
    if (responses[i].status === 200) ok++;
  }
  check(null, {
    'tf poll batch mostly ok': () => ok >= Math.floor(responses.length * 0.5),
  });
  return reqs.length;
}

/**
 * Parallel GETs for full tf-stress stack.
 * @param {import('../config.js').LoadConfig} cfg
 * @param {ReturnType<import('./urls.js').tfUrls>} urls
 * @param {import('./plan.js').TfPlan} plan
 * @returns {number} number of requests in the batch
 */
export function tfPollBatch(cfg, urls, plan) {
  return tfPollTargets(cfg, allPollTargets(urls, plan));
}

/**
 * Block until PUT (and GET) succeed for a resource. Used for slow workspace bootstrap.
 * @param {string} url
 * @param {object} body
 * @param {import('../config.js').LoadConfig} cfg
 * @param {string} label
 * @param {string} resource
 * @returns {boolean}
 */
export function tfPutUntilCreated(url, body, cfg, label, resource) {
  const payload = JSON.stringify(body);
  const pWrite = params(cfg, { tags: { name: 'tf_write', resource } });
  const pRead = params(cfg, { tags: { name: 'tf_poll', resource } });
  const maxAttempts = Number(__ENV.TF_WORKSPACE_WAIT_ATTEMPTS || '40');
  const pauseS = Number(__ENV.TF_WORKSPACE_WAIT_PAUSE_S || '2');

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    const putRes = http.put(url, payload, pWrite);
    recordResponse(putRes, { name: 'tf_write', resource });
    if (putRes.status === 200 || putRes.status === 201) {
      // Confirm readable before proceeding (avoids racing next creates).
      for (let g = 0; g < 10; g++) {
        const getRes = http.get(url, pRead);
        recordResponse(getRes, { name: 'tf_poll', resource });
        if (getRes.status === 200) {
          check(null, { [`${label} created`]: () => true });
          return true;
        }
        sleep(pauseS);
      }
    }
    if (
      putRes.status === 429 ||
      putRes.status === 500 ||
      putRes.status === 503 ||
      putRes.status === 200
    ) {
      sleep(pauseS * attempt * 0.5);
      continue;
    }
    logIfUnexpected(putRes, 200, `${label} attempt ${attempt}`);
    sleep(pauseS);
  }
  check(null, { [`${label} created`]: () => false });
  console.error(`${label}: failed to create after ${maxAttempts} attempts`);
  return false;
}

/**
 * True if the API says the workspace's K8s namespace is not ready yet
 * (still creating, gone, or Terminating after a prior destroy).
 * @param {import('k6/http').RefinedResponse} res
 */
export function isWorkspaceNamespaceNotReady(res) {
  if (!res || (res.status !== 404 && res.status !== 403 && res.status !== 500)) {
    return false;
  }
  const body = typeof res.body === 'string' ? res.body : String(res.body || '');
  return (
    body.indexOf('not found') !== -1 ||
    body.indexOf('being terminated') !== -1 ||
    body.indexOf('namespaces') !== -1
  );
}

/**
 * PUT network until the workspace child namespace accepts creates.
 * Workspace CR can be GET-able while its K8s namespace is missing/Terminating.
 *
 * @returns {import('k6/http').RefinedResponse}
 */
export function tfPutNetworkWhenNamespaceReady(cfg, urls, plan) {
  const url = urls.network(plan.network);
  const body = networkBody(plan.networkCidr);
  const payload = JSON.stringify(body);
  const p = params(cfg, { tags: { name: 'tf_write', resource: 'network' } });
  const maxAttempts = Number(__ENV.TF_NS_READY_ATTEMPTS || '60');
  const pauseS = Number(__ENV.TF_NS_READY_PAUSE_S || '2');

  let res;
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    res = http.put(url, payload, p);
    recordResponse(res, { name: 'tf_write', resource: 'network' });
    if (res.status === 200 || res.status === 201) {
      checkStatus(res, 200, `put network ${plan.network}`);
      return res;
    }
    if (isWorkspaceNamespaceNotReady(res) || res.status === 429 || res.status === 503) {
      if (attempt === 1 || attempt % 5 === 0) {
        console.log(
          `tf-net-storage: wait for workspace NS ready (${plan.workspace}) ` +
            `attempt ${attempt}/${maxAttempts} status=${res.status}`,
        );
      }
      sleep(pauseS);
      continue;
    }
    // Other errors (validation, etc.) — stop early.
    break;
  }
  logIfUnexpected(res, 200, `put network ${plan.network}`);
  checkStatus(res, 200, `put network ${plan.network}`);
  return res;
}

/**
 * Apply only network + block storages (workspace must already exist).
 * Waits for workspace K8s namespace readiness before creating children.
 * @returns {{ network: boolean, blockStorages: string[] }}
 */
export function tfApplyNetStorage(cfg, urls, plan) {
  const created = { network: false, blockStorages: [] };

  const netRes = tfPutNetworkWhenNamespaceReady(cfg, urls, plan);
  created.network = netRes.status === 200 || netRes.status === 201;
  if (!created.network) {
    return created;
  }

  for (let i = 0; i < plan.blockStorages.length; i++) {
    const name = plan.blockStorages[i];
    const sizeGB = 1 + (i % 3);
    const res = tfPut(
      urls.blockStorage(name),
      blockStorageBody(sizeGB),
      cfg,
      `put block-storage ${name}`,
      'block-storage',
    );
    if (res.status === 200 || res.status === 201) {
      created.blockStorages.push(name);
    }
  }
  return created;
}

/**
 * Destroy network + block storages (+ optional workspace).
 */
export function tfDestroyNetStorage(cfg, urls, plan, opts) {
  const destroyWorkspace = !opts || opts.destroyWorkspace !== false;
  const delOne = (url, label, resource) => {
    const res = http.del(url, null, params(cfg, { tags: { name: 'tf_delete', resource } }));
    recordResponse(res, { name: 'tf_delete', resource });
    check(res, {
      [`${label} delete ok`]: (r) =>
        r.status === 202 || r.status === 404 || r.status === 200,
    });
  };

  for (let i = plan.blockStorages.length - 1; i >= 0; i--) {
    delOne(
      urls.blockStorage(plan.blockStorages[i]),
      `del block-storage ${plan.blockStorages[i]}`,
      'block-storage',
    );
  }
  delOne(urls.network(plan.network), `del network ${plan.network}`, 'network');
  if (destroyWorkspace) {
    delOne(urls.workspace(), `del workspace ${plan.workspace}`, 'workspace');
  }
}

/**
 * Fixed polls over network/storage targets until deadline.
 */
export function tfFixedPollNetStorageUntil(cfg, urls, plan, created, deadlineMs) {
  let rounds = 0;
  while (Date.now() < deadlineMs) {
    const targets = netStoragePollTargets(urls, plan, { created });
    // Ensure ≥10 concurrent requests per batch (pad if needed).
    tfPollTargets(cfg, targets);
    rounds++;
    const wait = plan.pollIntervalS + plan.rnd() * plan.pollJitterS;
    const remaining = (deadlineMs - Date.now()) / 1000;
    if (remaining <= 0) break;
    sleep(Math.min(wait, remaining));
  }
  check(null, {
    'tf net-storage poll ran at least one round': () => rounds >= 1,
  });
  return rounds;
}

/**
 * Slowly create one workspace per VU index in setup (serial, blocking).
 * Names include runId so they never collide with Terminating namespaces
 * from a previous destroy of the same fixed names.
 *
 * @param {import('../config.js').LoadConfig} cfg
 * @param {number} userCount
 * @param {string} runId
 * @returns {{ workspaces: string[], ok: boolean, runId: string }}
 */
export function tfBootstrapWorkspacesSlow(cfg, userCount, runId) {
  const pauseS = Number(__ENV.TF_WORKSPACE_CREATE_PAUSE_S || '5');
  const rid = (runId || 'run').toString().replace(/[^a-zA-Z0-9-]/g, '').slice(0, 12);
  const workspaces = [];
  let ok = true;

  for (let vu = 1; vu <= userCount; vu++) {
    const tag = String(vu).padStart(2, '0');
    const workspace = `tfns-${rid}-w${tag}`;
    const url = `${cfg.baseUrlRegional}/providers/seca.workspace/v1/tenants/${cfg.tenant}/workspaces/${workspace}`;
    console.log(`tf-net-storage setup: creating workspace ${workspace} (slow)…`);
    const created = tfPutUntilCreated(
      url,
      workspaceBody(),
      cfg,
      `setup workspace ${workspace}`,
      'workspace',
    );
    if (!created) {
      ok = false;
    } else {
      workspaces.push(workspace);
    }
    if (vu < userCount) {
      sleep(pauseS);
    }
  }
  check(null, {
    'all workspaces created in setup': () => ok && workspaces.length === userCount,
  });
  return { workspaces, ok, runId: rid };
}

/**
 * Delete workspaces by name (setup-time cleanup). Used when
 * tfBootstrapWorkspacesSlow only partially succeeds, so a failed bootstrap
 * doesn't leak the workspaces it did manage to create into the next run.
 *
 * @param {import('../config.js').LoadConfig} cfg
 * @param {string[]} workspaceNames
 */
export function tfDeleteWorkspaces(cfg, workspaceNames) {
  for (const name of workspaceNames) {
    const url = tfUrls(cfg, name).workspace();
    const res = http.del(url, null, params(cfg, { tags: { name: 'tf_delete', resource: 'workspace' } }));
    recordResponse(res, { name: 'tf_delete', resource: 'workspace' });
    check(res, {
      [`cleanup: del workspace ${name} ok`]: (r) =>
        r.status === 202 || r.status === 200 || r.status === 404,
    });
  }
}

/**
 * Create full stack in dependency order.
 * @param {import('../config.js').LoadConfig} cfg
 * @param {ReturnType<import('./urls.js').tfUrls>} urls
 * @param {import('./plan.js').TfPlan} plan
 */
export function tfApply(cfg, urls, plan) {
  tfPut(urls.workspace(), workspaceBody(), cfg, `put workspace ${plan.workspace}`, 'workspace');

  tfPut(
    urls.network(plan.network),
    networkBody(plan.networkCidr),
    cfg,
    `put network ${plan.network}`,
    'network',
  );

  tfPut(
    urls.routeTable(plan.network, plan.routeTable),
    routeTableBody(plan.networkCidr),
    cfg,
    `put route-table ${plan.routeTable}`,
    'route-table',
  );

  for (let i = 0; i < plan.subnets.length; i++) {
    tfPut(
      urls.subnet(plan.network, plan.subnets[i]),
      subnetBody(plan.subnetCidrs[i], plan.routeTable, plan.zone),
      cfg,
      `put subnet ${plan.subnets[i]}`,
      'subnet',
    );
  }

  for (let i = 0; i < plan.blockStorages.length; i++) {
    const sizeGB = 1 + (i % 3);
    tfPut(
      urls.blockStorage(plan.blockStorages[i]),
      blockStorageBody(sizeGB),
      cfg,
      `put block-storage ${plan.blockStorages[i]}`,
      'block-storage',
    );
  }

  for (let i = 0; i < plan.instances.length; i++) {
    const vol = plan.blockStorages[i % plan.blockStorages.length];
    tfPut(
      urls.instance(plan.instances[i]),
      instanceBody(vol, plan.zone),
      cfg,
      `put instance ${plan.instances[i]}`,
      'instance',
    );
  }

  for (let i = 0; i < plan.nics.length; i++) {
    const sn = plan.subnets[i % plan.subnets.length];
    tfPut(
      urls.nic(plan.nics[i]),
      nicBody(sn),
      cfg,
      `put nic ${plan.nics[i]}`,
      'nic',
    );
  }
}

/**
 * Fixed poll phase until deadline (fixed_polls for dummy).
 * @param {import('../config.js').LoadConfig} cfg
 * @param {ReturnType<import('./urls.js').tfUrls>} urls
 * @param {import('./plan.js').TfPlan} plan
 * @param {number} deadlineMs Date.now() deadline
 */
export function tfFixedPollUntil(cfg, urls, plan, deadlineMs) {
  let rounds = 0;
  while (Date.now() < deadlineMs) {
    tfPollBatch(cfg, urls, plan);
    rounds++;
    const wait = plan.pollIntervalS + plan.rnd() * plan.pollJitterS;
    // Do not sleep past deadline by much.
    const remaining = (deadlineMs - Date.now()) / 1000;
    if (remaining <= 0) break;
    sleep(Math.min(wait, remaining));
  }
  check(null, {
    'tf poll ran at least one round': () => rounds >= 1,
  });
  return rounds;
}

/**
 * Reverse-order destroy.
 * @param {import('../config.js').LoadConfig} cfg
 * @param {ReturnType<import('./urls.js').tfUrls>} urls
 * @param {import('./plan.js').TfPlan} plan
 */
export function tfDestroy(cfg, urls, plan) {
  const delOne = (url, label, resource) => {
    const res = http.del(url, null, params(cfg, { tags: { name: 'tf_delete', resource } }));
    recordResponse(res, { name: 'tf_delete', resource });
    // 202 accepted, 404 already gone, 200 some stacks
    check(res, {
      [`${label} delete ok`]: (r) =>
        r.status === 202 || r.status === 404 || r.status === 200,
    });
    if (res.status !== 202 && res.status !== 404 && res.status !== 200) {
      logIfUnexpected(res, [202, 404, 200], label);
    }
  };

  for (let i = plan.nics.length - 1; i >= 0; i--) {
    delOne(urls.nic(plan.nics[i]), `del nic ${plan.nics[i]}`, 'nic');
  }
  for (let i = plan.instances.length - 1; i >= 0; i--) {
    delOne(urls.instance(plan.instances[i]), `del instance ${plan.instances[i]}`, 'instance');
  }
  for (let i = plan.blockStorages.length - 1; i >= 0; i--) {
    delOne(
      urls.blockStorage(plan.blockStorages[i]),
      `del block-storage ${plan.blockStorages[i]}`,
      'block-storage',
    );
  }
  for (let i = plan.subnets.length - 1; i >= 0; i--) {
    delOne(
      urls.subnet(plan.network, plan.subnets[i]),
      `del subnet ${plan.subnets[i]}`,
      'subnet',
    );
  }
  delOne(
    urls.routeTable(plan.network, plan.routeTable),
    `del route-table ${plan.routeTable}`,
    'route-table',
  );
  delOne(urls.network(plan.network), `del network ${plan.network}`, 'network');
  delOne(urls.workspace(), `del workspace ${plan.workspace}`, 'workspace');
}
