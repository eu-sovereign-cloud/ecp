// Count HTTP responses by class (2xx/3xx/4xx/5xx) and by exact status code.
//
// Built-in http_req_failed is only a boolean rate. These counters show *which*
// codes were returned. Record via recordResponse() after every request (wired
// into lib/http.js and lib/tf/apply.js). Summary text is printed in handleSummary.
//
// Note: k6 2.x end-of-test summary often collapses *tagged* custom counters into
// one series, so we use separate untagged counters (reliable in handleSummary).

import { Counter } from 'k6/metrics';

const classCounters = {
  '2xx': new Counter('http_2xx'),
  '3xx': new Counter('http_3xx'),
  '4xx': new Counter('http_4xx'),
  '5xx': new Counter('http_5xx'),
  '0_network': new Counter('http_0_network'),
  other: new Counter('http_status_other_class'),
};

// Exact codes we care about for SECA / k8s gateway load tests.
const codeCounters = {
  200: new Counter('http_status_200'),
  201: new Counter('http_status_201'),
  202: new Counter('http_status_202'),
  400: new Counter('http_status_400'),
  401: new Counter('http_status_401'),
  403: new Counter('http_status_403'),
  404: new Counter('http_status_404'),
  409: new Counter('http_status_409'),
  429: new Counter('http_status_429'),
  500: new Counter('http_status_500'),
  502: new Counter('http_status_502'),
  503: new Counter('http_status_503'),
};
const codeOther = new Counter('http_status_other');

/**
 * @param {number} status
 * @returns {string}
 */
export function statusClass(status) {
  if (status >= 200 && status < 300) return '2xx';
  if (status >= 300 && status < 400) return '3xx';
  if (status >= 400 && status < 500) return '4xx';
  if (status >= 500 && status < 600) return '5xx';
  if (status === 0) return '0_network';
  return 'other';
}

/**
 * Record one HTTP response for end-of-test breakdown.
 * @param {import('k6/http').RefinedResponse|null|undefined} res
 * @param {{ name?: string, resource?: string }} [_tags] ignored (kept for call-site compat)
 */
export function recordResponse(res, _tags) {
  if (!res) return;
  const status = typeof res.status === 'number' ? res.status : 0;
  const cls = statusClass(status);
  const cc = classCounters[cls] || classCounters.other;
  cc.add(1);

  const codeC = codeCounters[status];
  if (codeC) {
    codeC.add(1);
  } else {
    codeOther.add(1);
  }
}

/**
 * @param {object} metrics handleSummary data.metrics
 * @param {string} name
 * @returns {number}
 */
function metricCount(metrics, name) {
  const m = metrics[name];
  if (!m || !m.values) return 0;
  if (m.values.count !== undefined) return Number(m.values.count);
  return 0;
}

/**
 * Format status counters from handleSummary `data` for stdout.
 * @param {object} data k6 summary data
 * @returns {string}
 */
export function formatStatusBreakdown(data) {
  const metrics = (data && data.metrics) || {};
  const lines = ['', 'HTTP status breakdown:'];

  const classes = [
    ['2xx', 'http_2xx'],
    ['3xx', 'http_3xx'],
    ['4xx', 'http_4xx'],
    ['5xx', 'http_5xx'],
    ['0_network', 'http_0_network'],
    ['other', 'http_status_other_class'],
  ];
  let total = 0;
  for (const [label, key] of classes) {
    const n = metricCount(metrics, key);
    if (n > 0) {
      lines.push(`  ${label}: ${n}`);
      total += n;
    }
  }

  const codes = [200, 201, 202, 400, 401, 403, 404, 409, 429, 500, 502, 503];
  const codeLines = [];
  for (const code of codes) {
    const n = metricCount(metrics, `http_status_${code}`);
    if (n > 0) codeLines.push(`    ${code}: ${n}`);
  }
  const otherN = metricCount(metrics, 'http_status_other');
  if (otherN > 0) codeLines.push(`    other: ${otherN}`);

  if (codeLines.length > 0) {
    lines.push('  by code:');
    lines.push(...codeLines);
  }

  if (total === 0 && codeLines.length === 0) {
    lines.push('  (no samples recorded)');
  }
  lines.push('');
  return lines.join('\n');
}
