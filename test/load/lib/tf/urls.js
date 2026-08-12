// SECA regional URL builders for Terraform-like journeys.

/**
 * @param {import('../config.js').LoadConfig} cfg
 * @param {string} workspace
 */
export function tfUrls(cfg, workspace) {
  const t = cfg.tenant;
  const base = cfg.baseUrlRegional;
  const ws = `${base}/providers/seca.workspace/v1/tenants/${t}/workspaces/${workspace}`;
  const netP = `${base}/providers/seca.network/v1/tenants/${t}/workspaces/${workspace}`;
  const storP = `${base}/providers/seca.storage/v1/tenants/${t}/workspaces/${workspace}`;
  const compP = `${base}/providers/seca.compute/v1/tenants/${t}/workspaces/${workspace}`;

  return {
    workspace: () => ws,
    network: (name) => `${netP}/networks/${name}`,
    routeTable: (net, name) => `${netP}/networks/${net}/route-tables/${name}`,
    subnet: (net, name) => `${netP}/networks/${net}/subnets/${name}`,
    nic: (name) => `${netP}/nics/${name}`,
    blockStorage: (name) => `${storP}/block-storages/${name}`,
    instance: (name) => `${compP}/instances/${name}`,
  };
}
