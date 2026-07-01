#!/bin/bash
# report.sh — generate a benchmark latency report from a deployed gateway.
#
# Usage:
#   AUTHZ_IMPL=cached  make bench  # runs the load workload
#   AUTHZ_IMPL=cached  make report # scrapes /metrics and generates REPORT.md
#
# To compare two implementations:
#   AUTHZ_IMPL=cached  make bench
#   AUTHZ_IMPL=cached  make report SNAP_FILE=report/snap-cached.txt
#   # redeploy with AUTHZ_IMPL=direct (e.g. restart the gateway pod with AUTHZ_IMPL=direct)
#   AUTHZ_IMPL=direct  make bench
#   AUTHZ_IMPL=direct  make report SNAP_FILE=report/snap-direct.txt
#   go run ./cmd/benchreport \
#       --impl=cached --metrics-file=report/snap-cached.txt \
#       --impl=direct --metrics-file=report/snap-direct.txt \
#       --out=report/REPORT.md
#
# Environment variables:
#   GATEWAY_GLOBAL_PORT  — local port where gateway-global /metrics is reachable (default: auto via kubectl port-forward)
#   SNAP_FILE            — output file for the metrics snapshot (default: report/snap.txt)
#   IMPL_TAG             — implementation label in the report (default: value of AUTHZ_IMPL or "default")
#   OUT_FILE             — final report markdown file (default: report/REPORT.md)
set -euo pipefail

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
E2E_DIR="$(cd "${SCRIPTS_DIR}/.." && pwd)"

source "${SCRIPTS_DIR}/common.sh"

SNAP_FILE="${SNAP_FILE:-${E2E_DIR}/report/snap.txt}"
IMPL_TAG="${IMPL_TAG:-${AUTHZ_IMPL:-default}}"
OUT_FILE="${OUT_FILE:-${E2E_DIR}/report/REPORT.md}"

mkdir -p "${E2E_DIR}/report"

# Determine metrics endpoint.
if [ -n "${GATEWAY_GLOBAL_PORT:-}" ]; then
    METRICS_URL="http://localhost:${GATEWAY_GLOBAL_PORT}/metrics"
else
    echo "==> Detecting gateway-global port via kubectl..."
    # Start a temporary port-forward in the background.
    setup_kube_vars 2>/dev/null || true
    kubectl port-forward \
        -n "${NAMESPACE:-e2e-ecp}" \
        service/gateway-global-svc \
        8089:8080 &>/dev/null &
    PF_PID=$!
    sleep 2
    METRICS_URL="http://localhost:8089/metrics"
    trap 'kill ${PF_PID} 2>/dev/null || true' EXIT
fi

echo "==> Scraping metrics from ${METRICS_URL} → ${SNAP_FILE}"
curl -sf "${METRICS_URL}" -o "${SNAP_FILE}"

echo "==> Generating report: impl=${IMPL_TAG} snap=${SNAP_FILE} → ${OUT_FILE}"
cd "${E2E_DIR}"
go run ./cmd/benchreport \
    --impl="${IMPL_TAG}" \
    --metrics-file="${SNAP_FILE}" \
    --out="${OUT_FILE}"

echo "==> Report written to ${OUT_FILE}"
