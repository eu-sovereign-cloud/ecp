// Stepwise load journey: same light reads as smoke, ramped VUs.
//
// Profile (see options/stepwise.json):
//   warmup  10s × 1 VU  (~1 iter/s total when paced)
//   then 6 × 10s phases with VUs 2, 4, 6, 8, 10, 12 (+2 each step)
//
// Each VU paces to about 1 iteration/s so aggregate rate ≈ active VUs.
//
// Requires BASE_URL_GLOBAL, BASE_URL_REGIONAL. Run via:
//   make -C test/load stepwise

import http from 'k6/http';
import { check, sleep } from 'k6';

import { checkStatus, parseJSON } from '../lib/checks.js';
import { loadConfig } from '../lib/config.js';
import { get, logIfUnexpected } from '../lib/http.js';

export { handleSummary } from '../lib/summary.js';

// Target wall-clock per VU iteration (warmup: 1 VU → ~1 req-cycle/s).
const PACE_S = Number(__ENV.STEPWISE_PACE_S || '1');

export default function () {
  const cfg = loadConfig();
  const t0 = Date.now();

  // --- Liveness (no auth) ----------------------------------------------------
  const globalHealth = http.get(`${cfg.baseUrlGlobal}/healthz`);
  logIfUnexpected(globalHealth, 200, 'global /healthz');
  checkStatus(globalHealth, 200, 'global healthz');

  const regionalHealth = http.get(`${cfg.baseUrlRegional}/healthz`);
  logIfUnexpected(regionalHealth, 200, 'regional /healthz');
  checkStatus(regionalHealth, 200, 'regional healthz');

  // --- Regions (global) ------------------------------------------------------
  const regionsRes = get(cfg.regionsListURL(), cfg);
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

  // --- Workspaces list (regional, tenant-scoped) -----------------------------
  const wsRes = get(cfg.workspacesListURL(), cfg);
  logIfUnexpected(wsRes, 200, 'list workspaces');
  const wsOK = checkStatus(wsRes, 200, 'list workspaces');
  if (wsOK) {
    const body = parseJSON(wsRes);
    check(body, {
      'list workspaces has items array': (b) => b !== null && Array.isArray(b.items),
    });
  }

  // Pace: ~1 iteration per second per VU (warmup → 1/s total).
  const elapsedS = (Date.now() - t0) / 1000;
  if (elapsedS < PACE_S) {
    sleep(PACE_S - elapsedS);
  }
}
