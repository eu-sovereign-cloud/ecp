// Shared load-test configuration from environment variables.
// Real journeys call loadConfig() which requires gateway base URLs.
// Hello / self-checks may call loadConfig({ requireUrls: false }).

/**
 * @typedef {object} LoadConfig
 * @property {string} baseUrlGlobal
 * @property {string} baseUrlRegional
 * @property {string} tenant
 * @property {string} authUser
 * @property {string} authPass
 * @property {boolean} authEnabled
 * @property {string} systemNamespace
 * @property {string} regionProviderURL
 * @property {string} workspaceProviderURL
 * @property {function(string): string} workspaceURL path helper for a workspace name
 */

/**
 * @param {{ requireUrls?: boolean }} [opts]
 * @returns {LoadConfig}
 */
export function loadConfig(opts) {
  const requireUrls = !opts || opts.requireUrls !== false;

  const baseUrlGlobal = (__ENV.BASE_URL_GLOBAL || '').replace(/\/$/, '');
  const baseUrlRegional = (__ENV.BASE_URL_REGIONAL || '').replace(/\/$/, '');
  const tenant = __ENV.E2E_TENANT || 'test-tenant';
  const authUser = __ENV.AUTH_USER || 'admin';
  const authPass = __ENV.AUTH_PASS || 'e2e-admin-pass';
  const authEnabled = (__ENV.E2E_AUTH_ENABLED || 'true') !== 'false';
  const systemNamespace = __ENV.SYSTEM_NAMESPACE || 'e2e-ecp';

  if (requireUrls) {
    const missing = [];
    if (!baseUrlGlobal) missing.push('BASE_URL_GLOBAL');
    if (!baseUrlRegional) missing.push('BASE_URL_REGIONAL');
    if (missing.length > 0) {
      throw new Error(
        `missing required env: ${missing.join(', ')}. ` +
          'Point them at the global and regional gateways ' +
          '(e.g. after kubectl port-forward). See test/load/README.md.',
      );
    }
  }

  const regionProviderURL = baseUrlGlobal ? `${baseUrlGlobal}/providers/seca.region` : '';
  const workspaceProviderURL = baseUrlRegional
    ? `${baseUrlRegional}/providers/seca.workspace`
    : '';

  return {
    baseUrlGlobal,
    baseUrlRegional,
    tenant,
    authUser,
    authPass,
    authEnabled,
    systemNamespace,
    regionProviderURL,
    workspaceProviderURL,
    // PUT/GET/DELETE …/v1/tenants/{tenant}/workspaces/{name}
    workspaceURL(name) {
      return `${workspaceProviderURL}/v1/tenants/${tenant}/workspaces/${name}`;
    },
    workspacesListURL() {
      return `${workspaceProviderURL}/v1/tenants/${tenant}/workspaces`;
    },
    regionsListURL() {
      return `${regionProviderURL}/v1/regions`;
    },
  };
}
