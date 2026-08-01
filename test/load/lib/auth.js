// Dummy authenticator tokens for the test stack (AUTH_PLUGIN=dummy).
// Mirrors test/internal/authhelper MakeBearerToken: base64(JSON{username,password}).
// JWT is intentionally not implemented yet — mintToken() is the extension point.

import encoding from 'k6/encoding';

/**
 * @param {{ authUser: string, authPass: string, authEnabled: boolean }} cfg
 * @returns {string} bearer token material (no "Bearer " prefix), or '' if auth disabled
 */
export function mintToken(cfg) {
  if (!cfg.authEnabled) {
    return '';
  }
  // Dummy only for now. Future: branch on AUTH_PLUGIN / JWT_*.
  const payload = JSON.stringify({
    username: cfg.authUser,
    password: cfg.authPass,
  });
  return encoding.b64encode(payload);
}

/**
 * HTTP headers for gateway requests.
 * When auth is disabled, only Content-Type is set (matches unauthenticated e2e mode).
 *
 * @param {{ authUser: string, authPass: string, authEnabled: boolean }} cfg
 * @returns {Record<string, string>}
 */
export function authHeaders(cfg) {
  const headers = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  };
  if (!cfg.authEnabled) {
    return headers;
  }
  headers.Authorization = `Bearer ${mintToken(cfg)}`;
  return headers;
}
