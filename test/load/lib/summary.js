// Optional end-of-test HTML report via benc-uk/k6-reporter + handleSummary.
//
// Opt-in with REPORT_HTML=1 (default off). TABLES / checks / thresholds only —
// not time-series graphs. Graphed reports are the default via REPORT_DASHBOARD=1
// (k6 web dashboard export; wired in run-k6.sh — see README).
//
// Optional:
//   K6_HTML_REPORT   output path (default: reports/k6-report.html)
//   K6_REPORT_TITLE  HTML <title> (default: ECP k6 report)
//
// Journeys re-export this:
//   export { handleSummary } from '../lib/summary.js';

import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/latest/dist/bundle.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.1.0/index.js';

import { formatStatusBreakdown } from './status.js';

/**
 * @param {object} data k6 summary data
 * @returns {object} map of path → content for k6 to write
 */
export function handleSummary(data) {
  const reportOn =
    (__ENV.REPORT_HTML || '') === '1' ||
    (__ENV.REPORT_HTML || '').toLowerCase() === 'true';

  const statusBlock = formatStatusBreakdown(data);
  const files = {
    stdout:
      textSummary(data, { indent: ' ', enableColors: true }) + statusBlock,
  };

  if (!reportOn) {
    return files;
  }

  const out = __ENV.K6_HTML_REPORT || 'reports/k6-report.html';
  const title = __ENV.K6_REPORT_TITLE || 'ECP k6 report';
  files[out] = htmlReport(data, { title });
  console.log(`handleSummary: wrote HTML report → ${out}`);
  return files;
}
