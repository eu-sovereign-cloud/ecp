// Thin HTTP helpers: attach auth headers and optional unexpected-status logging.

import http from 'k6/http';

import { authHeaders } from './auth.js';

/**
 * @param {import('k6/http').RefinedResponse} res
 * @param {number|number[]} expected
 * @param {string} label
 */
export function logIfUnexpected(res, expected, label) {
  const want = Array.isArray(expected) ? expected : [expected];
  if (want.indexOf(res.status) === -1) {
    const body = typeof res.body === 'string' ? res.body : String(res.body);
    const snippet = body.length > 512 ? `${body.slice(0, 512)}…` : body;
    console.error(`${label}: status=${res.status} expected=${want.join('|')} body=${snippet}`);
  }
}

/**
 * @param {object} cfg loadConfig() result
 * @param {import('k6/http').Params} [params]
 * @returns {import('k6/http').Params}
 */
function withAuth(cfg, params) {
  const base = params || {};
  return {
    ...base,
    headers: {
      ...authHeaders(cfg),
      ...(base.headers || {}),
    },
  };
}

/**
 * @param {string} url
 * @param {object} cfg
 * @param {import('k6/http').Params} [params]
 */
export function get(url, cfg, params) {
  return http.get(url, withAuth(cfg, params));
}

/**
 * @param {string} url
 * @param {string|object|ArrayBuffer|null} body JSON object is stringified
 * @param {object} cfg
 * @param {import('k6/http').Params} [params]
 */
export function put(url, body, cfg, params) {
  const payload =
    body !== null && typeof body === 'object' && !(body instanceof ArrayBuffer)
      ? JSON.stringify(body)
      : body;
  return http.put(url, payload, withAuth(cfg, params));
}

/**
 * @param {string} url
 * @param {object} cfg
 * @param {import('k6/http').Params} [params]
 */
export function del(url, cfg, params) {
  return http.del(url, null, withAuth(cfg, params));
}
