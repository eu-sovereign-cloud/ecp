#!/usr/bin/env bash
# Apply Grafana dashboard JSON files in this directory to the monitoring
# ConfigMap used by the load-test Grafana deployment.
#
# Default target (matches the GKE load-test stack):
#   namespace: monitoring
#   configmap: grafana-dashboards
#   mount path in Grafana: /etc/grafana/dashboards
#
# Usage:
#   ./apply-dashboards.sh
#   NAMESPACE=monitoring CONFIGMAP=grafana-dashboards ./apply-dashboards.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DASH_DIR="${ROOT}/dashboards"
NAMESPACE="${NAMESPACE:-monitoring}"
CONFIGMAP="${CONFIGMAP:-grafana-dashboards}"

need() { command -v "$1" >/dev/null 2>&1 || { echo "required command not found: $1" >&2; exit 1; }; }
need kubectl

shopt -s nullglob
files=("${DASH_DIR}"/*.json)
if ((${#files[@]} == 0)); then
  echo "no dashboard JSON files in ${DASH_DIR}" >&2
  exit 1
fi

args=()
for f in "${files[@]}"; do
  name="$(basename "${f}")"
  args+=(--from-file="${name}=${f}")
  echo "  ${name}"
done

echo "Applying ${#files[@]} dashboard(s) to ${NAMESPACE}/${CONFIGMAP}"
kubectl -n "${NAMESPACE}" create configmap "${CONFIGMAP}" \
  "${args[@]}" \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart Grafana so the ConfigMap volume always remounts with new files.
# File watchers alone can lag behind ConfigMap projected volumes.
if kubectl -n "${NAMESPACE}" get deploy grafana >/dev/null 2>&1; then
  checksum="$(cat "${files[@]}" | shasum -a 256 | awk '{print $1}')"
  kubectl -n "${NAMESPACE}" annotate deploy grafana \
    "checksum/dashboards=${checksum}" --overwrite
  kubectl -n "${NAMESPACE}" rollout restart deploy/grafana
  kubectl -n "${NAMESPACE}" rollout status deploy/grafana --timeout=120s
  echo "Grafana restarted (checksum/dashboards=${checksum:0:12}…)."
fi

echo "Done. Open folder ECP → ECP Gateway Load Test."
