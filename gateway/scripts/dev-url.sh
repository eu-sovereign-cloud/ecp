#!/usr/bin/env bash
# Print the NodePort URL for a service in a 'gateway create-dev-clusters' KIND
# cluster (see setup-dev-clusters.sh) — reachable directly, no port-forward.
#
# Usage: dev-url.sh <cluster-name> <kubeconfig-path> <service-name>
set -euo pipefail

if [[ $# -ne 3 ]]; then
	echo "Usage: $0 <cluster-name> <kubeconfig-path> <service-name>" >&2
	exit 1
fi

CLUSTER_NAME="$1"
KUBECONFIG_PATH="$2"
SERVICE_NAME="$3"

if [[ ! -f "${KUBECONFIG_PATH}" ]]; then
	echo "dev-url: kubeconfig not found: ${KUBECONFIG_PATH} (run: make -C gateway create-dev-clusters)" >&2
	exit 1
fi

NODE_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${CLUSTER_NAME}-control-plane")
if [[ -z "${NODE_IP}" ]]; then
	echo "dev-url: could not determine node IP for cluster '${CLUSTER_NAME}' (is it running?)" >&2
	exit 1
fi

NODE_PORT=$(kubectl --kubeconfig "${KUBECONFIG_PATH}" get svc "${SERVICE_NAME}" -n default -o jsonpath='{.spec.ports[0].nodePort}')
if [[ -z "${NODE_PORT}" ]]; then
	echo "dev-url: could not determine NodePort for svc/${SERVICE_NAME} in cluster '${CLUSTER_NAME}'" >&2
	exit 1
fi

echo "http://${NODE_IP}:${NODE_PORT}"
