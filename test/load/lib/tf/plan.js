// Per-VU Terraform stack plan with random variation.

import { mulberry32, randFloat, randInt, seedForVu } from './rng.js';

/**
 * @typedef {object} TfPlan
 * @property {number} vu
 * @property {string} workspace
 * @property {string} network
 * @property {string} routeTable
 * @property {string[]} subnets
 * @property {string[]} blockStorages
 * @property {string[]} instances
 * @property {string[]} nics
 * @property {string} networkCidr
 * @property {string[]} subnetCidrs
 * @property {string} zone
 * @property {number} pollIntervalS
 * @property {number} pollJitterS
 * @property {number} journeyDurationS
 * @property {number} destroyBudgetS
 * @property {() => number} rnd
 */

/**
 * @param {number} vu k6 __VU (1-based)
 * @returns {TfPlan}
 */
export function buildPlan(vu) {
  const runId = __ENV.TF_RUN_ID || '';
  const rnd = mulberry32(seedForVu(vu, runId));

  const journeyDurationS = Number(__ENV.TF_JOURNEY_S || '300');
  const destroyBudgetS = Number(__ENV.TF_DESTROY_BUDGET_S || '45');

  // Base topology with variation.
  const nSubnets = randInt(rnd, 3, 5);
  const nBlock = randInt(rnd, 1, 3);
  const nInst = randInt(rnd, 15, 25);

  const tag = `${String(vu).padStart(2, '0')}`;
  const workspace = `tf-ws-${tag}`;
  const network = `tf-net-${tag}`;
  const routeTable = `tf-rt-${tag}`;

  const subnets = [];
  for (let i = 0; i < nSubnets; i++) {
    subnets.push(`tf-sn-${tag}-${i}`);
  }
  const blockStorages = [];
  for (let i = 0; i < nBlock; i++) {
    blockStorages.push(`tf-bs-${tag}-${i}`);
  }
  // Prefer enough volumes for instances: reuse round-robin if fewer volumes.
  const instances = [];
  const nics = [];
  for (let i = 0; i < nInst; i++) {
    instances.push(`tf-vm-${tag}-${i}`);
    nics.push(`tf-nic-${tag}-${i}`);
  }

  // Per-VU RFC1918 slice so CIDRs do not collide across users.
  const octet = 10 + (vu % 200);
  const networkCidr = `10.${octet}.0.0/16`;
  const subnetCidrs = [];
  for (let i = 0; i < nSubnets; i++) {
    subnetCidrs.push(`10.${octet}.${i + 1}.0/24`);
  }

  return {
    vu,
    workspace,
    network,
    routeTable,
    subnets,
    blockStorages,
    instances,
    nics,
    networkCidr,
    subnetCidrs,
    zone: __ENV.TF_ZONE || 'itbg-1',
    pollIntervalS: randFloat(rnd, 1.5, 3.0),
    pollJitterS: randFloat(rnd, 0.2, 1.0),
    journeyDurationS,
    destroyBudgetS,
    rnd,
  };
}

/**
 * All resource URLs for parallel poll (GET).
 * @param {ReturnType<import('./urls.js').tfUrls>} urls
 * @param {TfPlan} plan
 * @returns {{url: string, resource: string}[]}
 */
export function allPollTargets(urls, plan) {
  const out = [{ url: urls.workspace(), resource: 'workspace' }];
  out.push({ url: urls.network(plan.network), resource: 'network' });
  out.push({
    url: urls.routeTable(plan.network, plan.routeTable),
    resource: 'route-table',
  });
  for (const s of plan.subnets) {
    out.push({ url: urls.subnet(plan.network, s), resource: 'subnet' });
  }
  for (const b of plan.blockStorages) {
    out.push({ url: urls.blockStorage(b), resource: 'block-storage' });
  }
  for (const inst of plan.instances) {
    out.push({ url: urls.instance(inst), resource: 'instance' });
  }
  for (const n of plan.nics) {
    out.push({ url: urls.nic(n), resource: 'nic' });
  }
  return out;
}

/**
 * Lighter plan: workspace + 1 network + N block storages only.
 * Names include runId so a new run never reuses a workspace whose K8s
 * namespace is still Terminating from a previous destroy.
 *
 * @param {number} vu
 * @param {string} [runId] from setup (required for unique names across runs)
 * @returns {TfPlan}
 */
export function buildNetStoragePlan(vu, runId) {
  const rid = (runId || __ENV.TF_RUN_ID || 'run').toString().replace(/[^a-zA-Z0-9-]/g, '').slice(0, 12);
  const rnd = mulberry32(seedForVu(vu, `ns:${rid}`));

  const journeyDurationS = Number(__ENV.TF_JOURNEY_S || '300');
  const destroyBudgetS = Number(__ENV.TF_DESTROY_BUDGET_S || '45');
  const nBlock = randInt(rnd, 1, 3);

  const tag = `${String(vu).padStart(2, '0')}`;
  // K8s resource names max 63 chars; keep short.
  const workspace = `tfns-${rid}-w${tag}`;
  const network = `tfns-${rid}-n${tag}`;
  const blockStorages = [];
  for (let i = 0; i < nBlock; i++) {
    blockStorages.push(`tfns-${rid}-b${tag}-${i}`);
  }

  const octet = 40 + (vu % 150);
  const networkCidr = `10.${octet}.0.0/16`;

  return {
    vu,
    workspace,
    network,
    routeTable: '',
    subnets: [],
    blockStorages,
    instances: [],
    nics: [],
    networkCidr,
    subnetCidrs: [],
    zone: __ENV.TF_ZONE || 'itbg-1',
    pollIntervalS: randFloat(rnd, 1.5, 3.0),
    pollJitterS: randFloat(rnd, 0.2, 1.0),
    journeyDurationS,
    destroyBudgetS,
    rnd,
    runId: rid,
  };
}

/**
 * Poll targets for network + block-storage journey (plus workspace).
 * @param {ReturnType<import('./urls.js').tfUrls>} urls
 * @param {TfPlan} plan
 * @param {{ includeWorkspace?: boolean, created?: { network?: boolean, blockStorages?: string[] } }} [opts]
 */
export function netStoragePollTargets(urls, plan, opts) {
  const o = opts || {};
  const created = o.created || {};
  const out = [];
  if (o.includeWorkspace !== false) {
    out.push({ url: urls.workspace(), resource: 'workspace' });
  }
  // Only poll resources we successfully created (avoids 404 noise).
  if (created.network === true) {
    out.push({ url: urls.network(plan.network), resource: 'network' });
  }
  const bsList = Array.isArray(created.blockStorages)
    ? created.blockStorages
    : plan.blockStorages;
  for (const b of bsList) {
    out.push({ url: urls.blockStorage(b), resource: 'block-storage' });
  }
  return out;
}
