// Smoke journey: stack is reachable and light authenticated reads succeed.
//
// 1. GET /healthz on global + regional (no auth)
// 2. GET regions (authn-only provider on global)
// 3. GET workspaces for tenant (authz + tenant wiring on regional)
//
// Requires BASE_URL_GLOBAL, BASE_URL_REGIONAL. Run via:
//   make -C test/load smoke

import http from 'k6/http';
import { check } from 'k6';

import { checkStatus, parseJSON } from '../lib/checks.js';
import { loadConfig } from '../lib/config.js';
import { get, logIfUnexpected } from '../lib/http.js';

export { handleSummary } from '../lib/summary.js';

function hintTenantOrFixtures(status, tenant) {
  console.error(
    `smoke: list workspaces for tenant ${tenant} returned ${status}. ` +
      'If the tenant Namespace is missing: make -C test/load ensure-tenant. ' +
      'If RBAC/fixtures are missing: make -C test deploy-test-data ' +
      '(or kind-deploy-test-data / kind-stack). See test/load/README.md.',
  );
}

export default function () {
  const cfg = loadConfig(); // requires BASE_URL_*

  // --- 1. Liveness (no auth) -------------------------------------------------
  const globalHealth = http.get(`${cfg.baseUrlGlobal}/healthz`);
  logIfUnexpected(globalHealth, 200, 'global /healthz');
  checkStatus(globalHealth, 200, 'global healthz');

  const regionalHealth = http.get(`${cfg.baseUrlRegional}/healthz`);
  logIfUnexpected(regionalHealth, 200, 'regional /healthz');
  checkStatus(regionalHealth, 200, 'regional healthz');

  // --- 2. Regions (authn-only; needs bearer when auth enabled) ---------------
  const regionsURL = cfg.regionsListURL();
  const regionsRes = get(regionsURL, cfg);
  logIfUnexpected(regionsRes, 200, 'list regions');
  const regionsOK = checkStatus(regionsRes, 200, 'list regions');
  if (regionsOK) {
    const body = parseJSON(regionsRes);
    check(body, {
      'list regions has items array': (b) => b !== null && Array.isArray(b.items),
      'list regions items non-empty': (b) =>
        b !== null && Array.isArray(b.items) && b.items.length > 0,
    });
  }

  // --- 3. Workspaces list (tenant-scoped; proves authz + tenant ns) ----------
  const wsURL = cfg.workspacesListURL();
  const wsRes = get(wsURL, cfg);
  if (wsRes.status === 403 || wsRes.status === 404) {
    hintTenantOrFixtures(wsRes.status, cfg.tenant);
  }
  logIfUnexpected(wsRes, 200, 'list workspaces');
  const wsOK = checkStatus(wsRes, 200, 'list workspaces');
  if (wsOK) {
    const body = parseJSON(wsRes);
    check(body, {
      'list workspaces has items array': (b) => b !== null && Array.isArray(b.items),
    });
  }
}
