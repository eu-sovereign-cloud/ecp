#!/bin/bash
# Install both charts, from the published Dockerfiles, into a throwaway KIND
# cluster and assert the gateways actually came up configured.
#
# Why this exists: `helm template` proves a chart renders, not that what it
# renders works. The charts configure the gateways entirely through container
# args, and the released images are the bare binary — a values key that reaches
# no flag produces a healthy pod serving the wrong configuration, which is
# exactly how auth once ended up silently disabled on every install. The one
# assertion that catches that class of bug is a request: with auth.enabled=true
# an anonymous call must be rejected, and an authenticated one must succeed.
#
# The gateways are built from gateway/build/Dockerfile.* — the images `helm
# install` pulls. The delegator here runs the dummy plugin (the only self-
# contained one), built from csp/dummy/build/Dockerfile; dummy is dev/test-only
# and unpublished, so this is its debug image side-loaded like the e2e stack's.
#
# Usage: ci/scripts/chart-smoke.sh [cluster-name]
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLUSTER="${1:-chart-smoke}"
NAMESPACE=ecp-smoke
TAG=smoke
REGION=itbg-bergamo

cleanup() {
    kill "${PF_PID:-}" 2>/dev/null || true
    if [ "${KEEP_CLUSTER:-}" != "true" ]; then
        kind delete cluster --name "${CLUSTER}" >/dev/null 2>&1 || true
    fi
}
trap cleanup EXIT

cd "${REPO_ROOT}"

echo "==> Building the published images"
docker build -q -f gateway/build/Dockerfile.global   -t "ecp/gateway-global:${TAG}"   . >/dev/null
docker build -q -f gateway/build/Dockerfile.regional -t "ecp/gateway-regional:${TAG}" . >/dev/null
docker build -q -f csp/dummy/build/Dockerfile -t "ecp/delegator-dummy:${TAG}" . >/dev/null

echo "==> Creating KIND cluster ${CLUSTER}"
ci/scripts/kind-cgroup-preflight.sh
kind create cluster --name "${CLUSTER}"
kind load docker-image --name "${CLUSTER}" \
    "ecp/gateway-global:${TAG}" "ecp/gateway-regional:${TAG}" "ecp/delegator-dummy:${TAG}"

echo "==> helm install ecp"
helm install ecp charts/ecp \
    --namespace "${NAMESPACE}" --create-namespace \
    --set "gatewayRegional.region=${REGION}" \
    --set "gatewayGlobal.image.repository=ecp/gateway-global" \
    --set "gatewayGlobal.image.tag=${TAG}" \
    --set "gatewayRegional.image.repository=ecp/gateway-regional" \
    --set "gatewayRegional.image.tag=${TAG}" \
    --set auth.enabled=true \
    --set "auth.dummyUsers.users.smoke=smoke-pass" \
    --wait --timeout 5m

echo "==> helm install ecp-delegator"
helm install ecp-delegator charts/delegator \
    --namespace "${NAMESPACE}" \
    --set plugin=dummy \
    --set "image.repository=ecp/delegator-dummy" \
    --set "image.tag=${TAG}" \
    --wait --timeout 5m

# seca.region is served authn-only (it is tenant-less by spec), so it answers
# for any authenticated caller — which makes it the endpoint that isolates
# "did the auth flags arrive" from "is RBAC configured".
echo "==> Probing the global gateway"
kubectl -n "${NAMESPACE}" port-forward svc/ecp-gateway-global 18080:80 >/dev/null 2>&1 &
PF_PID=$!
for _ in $(seq 30); do
    curl -so /dev/null "http://localhost:18080/providers/seca.region/v1/regions" && break
    sleep 1
done

URL="http://localhost:18080/providers/seca.region/v1/regions"
# The dummy authenticator's bearer token is base64(JSON{username,password}) —
# see test/internal/authhelper.
TOKEN=$(printf '{"username":"smoke","password":"smoke-pass"}' | base64 -w0)

anon=$(curl -so /dev/null -w '%{http_code}' "${URL}")
auth=$(curl -so /dev/null -w '%{http_code}' -H "Authorization: Bearer ${TOKEN}" "${URL}")

echo "    anonymous: ${anon} (want 401)"
echo "    authenticated: ${auth} (want 200)"

if [ "${anon}" != "401" ] || [ "${auth}" != "200" ]; then
    echo "FAIL: auth.enabled=true did not reach the gateway" >&2
    kubectl -n "${NAMESPACE}" get deploy -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.template.spec.containers[0].args}{"\n"}{end}' >&2
    kubectl -n "${NAMESPACE}" logs -l app=gateway-global --tail=50 >&2 || true
    exit 1
fi

echo "==> OK: both charts install and the gateway enforces the configured auth"
