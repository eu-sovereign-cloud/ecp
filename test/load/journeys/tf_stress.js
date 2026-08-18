// tf-stress: Terraform-shaped apply × 10 concurrent users (one workspace each).
//
// Isolation: multi-workspace under E2E_TENANT (tf-ws-01 … tf-ws-10).
// Wait mode: fixed_polls — fill ~TF_JOURNEY_S (default 300s) with parallel GETs.
// Always destroy in reverse order at the end.
//
// Stack (base + variation): workspace, network, route-table, 3–5 subnets,
// 1–3 block-storages, 15–25 instances, 1 NIC per instance.
//
// Concurrency: each poll round uses http.batch over the full resource set
// (~40+ GETs per VU). With 10 VUs that is ≫ 10 concurrent API requests.
//
//   make -C test/load tf-stress

import { check, sleep } from 'k6';

import { loadConfig } from '../lib/config.js';
import { get, logIfUnexpected } from '../lib/http.js';
import {
  tfApply,
  tfDestroy,
  tfExpectStatuses,
  tfFixedPollUntil,
  tfPollBatch,
} from '../lib/tf/apply.js';
import { buildPlan } from '../lib/tf/plan.js';
import { tfUrls } from '../lib/tf/urls.js';

export { handleSummary } from '../lib/summary.js';

export function setup() {
  const cfg = loadConfig();
  // Light global probe so misconfig fails before 10 VUs hammer the API.
  const res = get(cfg.regionsListURL(), cfg);
  logIfUnexpected(res, 200, 'setup regions list');
  check(res, { 'setup regions reachable': (r) => r.status === 200 });
  return {
    startedAt: Date.now(),
  };
}

export default function () {
  tfExpectStatuses();

  const cfg = loadConfig();
  const plan = buildPlan(__VU);
  const urls = tfUrls(cfg, plan.workspace);

  const t0 = Date.now();
  const deadlineApplyPoll = t0 + (plan.journeyDurationS - plan.destroyBudgetS) * 1000;

  console.log(
    `tf-stress VU=${__VU} workspace=${plan.workspace} ` +
      `subnets=${plan.subnets.length} volumes=${plan.blockStorages.length} ` +
      `instances=${plan.instances.length} journey=${plan.journeyDurationS}s`,
  );

  // Optional small stagger so creates do not all hit the same millisecond.
  sleep(plan.rnd() * 1.5);

  // --- Apply (writes) --------------------------------------------------------
  tfApply(cfg, urls, plan);

  // Immediate multi-GET batch (contribute to ≥10 concurrent across VUs).
  const batchN = tfPollBatch(cfg, urls, plan);
  check(null, {
    'tf poll batch size >= 10': () => batchN >= 10,
  });

  // --- Fixed polls until ~5 min minus destroy budget -------------------------
  if (Date.now() < deadlineApplyPoll) {
    tfFixedPollUntil(cfg, urls, plan, deadlineApplyPoll);
  }

  // --- Always destroy --------------------------------------------------------
  tfDestroy(cfg, urls, plan);

  const elapsedS = (Date.now() - t0) / 1000;
  check(null, {
    'tf journey roughly multi-minute': () => elapsedS >= plan.journeyDurationS * 0.5,
  });
}
