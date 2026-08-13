// Shared k6 check helpers for journeys.

import { check } from 'k6';

/**
 * Assert response status is exactly expected.
 *
 * @param {import('k6/http').RefinedResponse} res
 * @param {number} expected
 * @param {string} name check name prefix
 * @returns {boolean}
 */
export function checkStatus(res, expected, name) {
  return check(res, {
    [`${name} status is ${expected}`]: (r) => r.status === expected,
  });
}

/**
 * Assert response status is one of the expected codes.
 *
 * @param {import('k6/http').RefinedResponse} res
 * @param {number[]} expected
 * @param {string} name
 * @returns {boolean}
 */
export function checkStatusOneOf(res, expected, name) {
  return check(res, {
    [`${name} status in ${expected.join('|')}`]: (r) => expected.indexOf(r.status) !== -1,
  });
}

/**
 * Parse JSON body; returns null on failure.
 *
 * @param {import('k6/http').RefinedResponse} res
 * @returns {object|null}
 */
export function parseJSON(res) {
  try {
    return res.json();
  } catch (_) {
    return null;
  }
}
