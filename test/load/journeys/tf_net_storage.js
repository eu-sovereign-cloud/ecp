// tf-net-storage: light Terraform-like journey (workspace → network + block storage).
//
// 1. setup(): create 10 workspaces VERY SLOWLY and serially with a unique runId
//    in each name (avoids Terminating K8s namespaces from prior runs). Each
//    workspace must PUT+GET 200 before the next; setup aborts if any fail.
// 2. Each of 10 VUs: confirm workspace, create 1 network + 1–3 block storages
//    only — network PUT waits until the workspace child namespace is ready
//    (not missing / not Terminating). Then fixed_poll GETs, always destroy.
//
//   make -C test/load tf-net-storage

import {check, sleep} from 'k6';

import {loadConfig} from '../lib/config.js';
import {get, logIfUnexpected} from '../lib/http.js';
import {
  tfApplyNetStorage,
  tfBootstrapWorkspacesSlow,
  tfDeleteWorkspaces,
  tfDestroyNetStorage,
  tfExpectStatuses,
  tfFixedPollNetStorageUntil,
  tfPollTargets,
} from '../lib/tf/apply.js';
import {buildNetStoragePlan, netStoragePollTargets} from '../lib/tf/plan.js';
import {tfUrls} from '../lib/tf/urls.js';

export { handleSummary } from '../lib/summary.js';

const USER_COUNT = Number(__ENV.TF_USERS || '10');

export function setup() {
  tfExpectStatuses();
  const cfg = loadConfig();

  const regions = get(cfg.regionsListURL(), cfg);
  logIfUnexpected(regions, 200, 'setup regions list');
  check(regions, { 'setup regions reachable': (r) => r.status === 200 });

  // Unique per run so destroy leftovers (Terminating namespaces) cannot block us.
  const runId = (__ENV.TF_RUN_ID || `${Date.now()}`).toString().replace(/[^a-zA-Z0-9-]/g, '').slice(0, 12);
  console.log(`tf-net-storage setup: runId=${runId}`);

  const boot = tfBootstrapWorkspacesSlow(cfg, USER_COUNT, runId);
  if (!boot.ok) {
    if (boot.workspaces.length > 0) {
      console.error(
        `tf-net-storage setup: bootstrap failed; deleting ${boot.workspaces.length} ` +
          'partially-created workspace(s) before aborting',
      );
      tfDeleteWorkspaces(cfg, boot.workspaces);
    }
    throw new Error(
      'tf-net-storage setup: not all workspaces created; aborting before net/storage phase',
    );
  }

  return {
    startedAt: Date.now(),
    workspaces: boot.workspaces,
    runId: boot.runId,
  };
}

export default function (data) {
  tfExpectStatuses();

  const cfg = loadConfig();
  const runId = (data && data.runId) || __ENV.TF_RUN_ID || 'run';
  const plan = buildNetStoragePlan(__VU, runId);
  const urls = tfUrls(cfg, plan.workspace);

  const expected = data && data.workspaces ? data.workspaces : [];
  if (expected.indexOf(plan.workspace) === -1) {
    console.error(`tf-net-storage VU=${__VU}: workspace ${plan.workspace} missing from setup`);
    check(null, { 'workspace available from setup': () => false });
    return;
  }

  // Re-confirm workspace CR is GET-able before child creates.
  let ready = false;
  for (let i = 0; i < 15; i++) {
    const res = get(urls.workspace(), cfg);
    if (res.status === 200) {
      ready = true;
      break;
    }
    sleep(1);
  }
  check(null, { 'workspace GET ok before apply': () => ready });
  if (!ready) {
    console.error(`tf-net-storage VU=${__VU}: workspace ${plan.workspace} not readable; skip`);
    return;
  }

  const t0 = Date.now();
  const deadline = t0 + (plan.journeyDurationS - plan.destroyBudgetS) * 1000;

  console.log(
    `tf-net-storage VU=${__VU} workspace=${plan.workspace} ` +
      `network=1 volumes=${plan.blockStorages.length} journey=${plan.journeyDurationS}s`,
  );

  sleep(plan.rnd() * 1.0);

  // Network create waits until workspace K8s namespace is ready (not only CR).
  const created = tfApplyNetStorage(cfg, urls, plan);
  check(null, {
    'network created': () => created.network === true,
    'at least one block storage created': () => created.blockStorages.length > 0,
  });

  if (!created.network && created.blockStorages.length === 0) {
    console.error(`tf-net-storage VU=${__VU}: no resources created; skip poll/destroy children`);
    // Still try to delete workspace so we do not leak CRs.
    tfDestroyNetStorage(cfg, urls, plan, { destroyWorkspace: true });
    return;
  }

  const targets = netStoragePollTargets(urls, plan, { created });
  const batchN = tfPollTargets(cfg, targets);
  check(null, {
    'tf net-storage poll batch size >= 10': () => batchN >= 10,
  });

  if (Date.now() < deadline) {
    tfFixedPollNetStorageUntil(cfg, urls, plan, created, deadline);
  }

  const destroyPlan = Object.assign({}, plan, {
    blockStorages: created.blockStorages.length ? created.blockStorages : plan.blockStorages,
  });
  tfDestroyNetStorage(cfg, urls, destroyPlan, { destroyWorkspace: true });
}
