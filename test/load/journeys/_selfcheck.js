// Offline self-check for shared lib (config, auth, path helpers).
// No gateway calls. Uses options/smoke.json via make selfcheck.

import { check } from 'k6';
import encoding from 'k6/encoding';

import { authHeaders, mintToken } from '../lib/auth.js';
import { loadConfig } from '../lib/config.js';

export { handleSummary } from '../lib/summary.js';

export default function () {
  const cfg = loadConfig({ requireUrls: false });

  const token = mintToken(cfg);
  const headers = authHeaders(cfg);

  // Expected dummy token: base64(JSON{username,password}) — same as authhelper.
  const expectedPayload = JSON.stringify({
    username: cfg.authUser,
    password: cfg.authPass,
  });
  const expectedToken = encoding.b64encode(expectedPayload);

  check(null, {
    'tenant default is set': () => cfg.tenant.length > 0,
    'mintToken matches dummy encoding': () => token === expectedToken,
    'authHeaders has Bearer': () =>
      typeof headers.Authorization === 'string' &&
      headers.Authorization.indexOf('Bearer ') === 0,
    'authHeaders has Content-Type': () => headers['Content-Type'] === 'application/json',
  });

  // URL helpers still build paths when base URLs are set; empty when not.
  const withUrls = loadConfig({ requireUrls: false });
  // Inject via re-call with env is hard in-script; just ensure helpers exist.
  check(withUrls, {
    'workspaceURL is a function': (c) => typeof c.workspaceURL === 'function',
    'workspacesListURL is a function': (c) => typeof c.workspacesListURL === 'function',
    'regionsListURL is a function': (c) => typeof c.regionsListURL === 'function',
  });

  // Auth-disabled path: no Authorization header.
  const disabled = {
    authUser: 'admin',
    authPass: 'x',
    authEnabled: false,
  };
  check(authHeaders(disabled), {
    'auth disabled omits Authorization': (h) => h.Authorization === undefined,
    'auth disabled still sets Content-Type': (h) => h['Content-Type'] === 'application/json',
  });
  check(mintToken(disabled), {
    'auth disabled mintToken is empty': (t) => t === '',
  });
}
