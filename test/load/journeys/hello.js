// No-network self-check: proves k6 can load shared lib and print config.
// Does not call any gateway (BASE_URL_* optional).

import { loadConfig } from '../lib/config.js';

export { handleSummary } from '../lib/summary.js';

export const options = {
  vus: 1,
  iterations: 1,
};

export default function () {
  const cfg = loadConfig({ requireUrls: false });
  console.log(
    JSON.stringify({
      tenant: cfg.tenant,
      authUser: cfg.authUser,
      authEnabled: cfg.authEnabled,
      systemNamespace: cfg.systemNamespace,
      baseUrlGlobal: cfg.baseUrlGlobal || '(unset)',
      baseUrlRegional: cfg.baseUrlRegional || '(unset)',
    }),
  );
}
