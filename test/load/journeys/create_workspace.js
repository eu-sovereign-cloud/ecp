// create_workspace journey: PUT create → GET read-back → DELETE cleanup.
//
// Gateway REST contract only by default (matches integration suite).
// Set WAIT_ACTIVE=1 to poll until status.state is Active (needs delegator).
//
// Requires BASE_URL_REGIONAL (and BASE_URL_GLOBAL for loadConfig). Run via:
//   make -C test/load create-workspace

import { check, sleep } from 'k6';

import { checkStatus, parseJSON } from '../lib/checks.js';
import { loadConfig } from '../lib/config.js';
import { del, get, logIfUnexpected, put } from '../lib/http.js';

const WAIT_ACTIVE = (__ENV.WAIT_ACTIVE || '0') === '1';
const ACTIVE_TIMEOUT_S = Number(__ENV.ACTIVE_TIMEOUT_S || '60');
const ACTIVE_POLL_S = Number(__ENV.ACTIVE_POLL_S || '2');

function hintTenantOrFixtures(status, tenant) {
  console.error(
    `create_workspace: request for tenant ${tenant} returned ${status}. ` +
      'If the tenant Namespace is missing: make -C test/load ensure-tenant. ' +
      'If RBAC/fixtures are missing: make -C test deploy-test-data ' +
      '(or kind-deploy-test-data / kind-stack). See test/load/README.md.',
  );
}

function waitForActive(cfg, url, name) {
  const deadline = Date.now() + ACTIVE_TIMEOUT_S * 1000;
  while (Date.now() < deadline) {
    const res = get(url, cfg);
    if (res.status === 200) {
      const body = parseJSON(res);
      const state = body && body.status && body.status.state;
      if (state === 'Active') {
        return true;
      }
    }
    sleep(ACTIVE_POLL_S);
  }
  console.error(
    `create_workspace: workspace ${name} did not become Active within ${ACTIVE_TIMEOUT_S}s ` +
      '(is the delegator running?).',
  );
  return false;
}

export function setup() {
  // Fail fast on missing URLs before VUs start; tenant ensure is a Make prereq.
  loadConfig();
  return {
    waitActive: WAIT_ACTIVE,
  };
}

export default function () {
  const cfg = loadConfig();
  const name = `k6-ws-${__VU}-${__ITER}-${Date.now()}`;
  const url = cfg.workspaceURL(name);

  try {
    // --- Create ---------------------------------------------------------------
    const createRes = put(url, {}, cfg);
    if (createRes.status === 403 || createRes.status === 404) {
      hintTenantOrFixtures(createRes.status, cfg.tenant);
    }
    logIfUnexpected(createRes, 200, `create workspace ${name}`);
    const created = checkStatus(createRes, 200, 'create workspace');
    if (!created) {
      return;
    }

    // --- Read-back ------------------------------------------------------------
    const getRes = get(url, cfg);
    logIfUnexpected(getRes, 200, `get workspace ${name}`);
    const got = checkStatus(getRes, 200, 'get workspace');
    if (got) {
      const body = parseJSON(getRes);
      check(body, {
        'get workspace metadata.name matches': (b) =>
          b !== null && b.metadata && b.metadata.name === name,
      });
    }

    // --- Optional Active wait -------------------------------------------------
    if (WAIT_ACTIVE) {
      const active = waitForActive(cfg, url, name);
      check(null, {
        'workspace reached Active': () => active,
      });
    }
  } finally {
    // Best-effort cleanup so failed checks still remove the resource.
    const delRes = del(url, cfg);
    // 202 Accepted is the success contract; 404 means already gone.
    if (delRes.status !== 202 && delRes.status !== 404) {
      logIfUnexpected(delRes, [202, 404], `delete workspace ${name}`);
    }
    check(delRes, {
      'delete workspace accepted or gone': (r) => r.status === 202 || r.status === 404,
    });
  }
}
