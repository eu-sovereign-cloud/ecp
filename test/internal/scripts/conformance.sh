#!/bin/bash
# Deploy the SECA conformance runner and execute secatest against a gateway.
#
# This is plugin-generic: whatever plugin the delegator was deployed with (dummy,
# aruba, ionos, ...) is what the conformance run exercises. Deploying the stack is
# `conformance-deploy`'s job (honouring CONFORMANCE_PLUGIN); the `conformance`
# target only runs this script, which stands up the conformance pod and runs
# secatest in it.
#
# Configuration comes from CONFORMANCE_* environment variables (see the Makefile
# defaults); all have sensible fallbacks for the in-cluster dummy setup.
set -eo pipefail

source "$(dirname "$0")/common.sh"
setup_env
source_config
setup_kube_vars
setup_registry_vars conformance

NAMESPACE="${CONFORMANCE_NAMESPACE:-e2e-ecp}"
DEPLOY_DIR="${SCRIPT_DIR}/../deploy/conformance"

# Defaults target the in-cluster global gateway with the dummy test-data tenant.
: "${CONFORMANCE_PROVIDER_REGION_V1:=http://${GATEWAY_GLOBAL_SVC}:80/providers/seca.region}"
: "${CONFORMANCE_PROVIDER_AUTHORIZATION_V1:=http://${GATEWAY_REGIONAL_SVC}:80/providers/seca.authorization}}"
: "${CONFORMANCE_AUTH_TOKEN:=test-token}"
: "${CONFORMANCE_TENANT:=test-tenant}"
: "${CONFORMANCE_REGION:=itbg-bergamo}"
: "${CONFORMANCE_SCENARIOS:=.*}"
: "${CONFORMANCE_USERS:=}"
: "${CONFORMANCE_CIDR:=}"
: "${CONFORMANCE_PUBLIC_IPS:=}"
: "${CONFORMANCE_RETRY_BASE_DELAY:=10}"
: "${CONFORMANCE_RETRY_BASE_INTERVAL:=10}"
: "${CONFORMANCE_RETRY_MAX_ATTEMPTS:=9}"

echo "Rendering conformance runner (image ${IMAGE_NAME}, scenarios ${CONFORMANCE_SCENARIOS})..."
YAML=$(kubectl kustomize "${DEPLOY_DIR}")
# Bash substitution (not sed) so values containing slashes, e.g. URLs, are safe.
YAML=${YAML//'##IMAGE_NAME##'/${IMAGE_NAME}}
YAML=${YAML//'##PROVIDER_REGION_V1##'/${CONFORMANCE_PROVIDER_REGION_V1}}
YAML=${YAML//'##PROVIDER_AUTHORIZATION_V1##'/${CONFORMANCE_PROVIDER_AUTHORIZATION_V1}}
YAML=${YAML//'##CLIENT_AUTH_TOKEN##'/${CONFORMANCE_AUTH_TOKEN}}
YAML=${YAML//'##CLIENT_TENANT##'/${CONFORMANCE_TENANT}}
YAML=${YAML//'##CLIENT_REGION##'/${CONFORMANCE_REGION}}
YAML=${YAML//'##SCENARIOS_FILTER##'/${CONFORMANCE_SCENARIOS}}
YAML=${YAML//'##SCENARIOS_USERS##'/${CONFORMANCE_USERS}}
YAML=${YAML//'##SCENARIOS_CIDR##'/${CONFORMANCE_CIDR}}
YAML=${YAML//'##SCENARIOS_PUBLIC_IPS##'/${CONFORMANCE_PUBLIC_IPS}}
YAML=${YAML//'##RETRY_BASE_DELAY##'/${CONFORMANCE_RETRY_BASE_DELAY}}
YAML=${YAML//'##RETRY_BASE_INTERVAL##'/${CONFORMANCE_RETRY_BASE_INTERVAL}}
YAML=${YAML//'##RETRY_MAX_ATTEMPTS##'/${CONFORMANCE_RETRY_MAX_ATTEMPTS}}
if [[ -n "$USE_KIND" && "$USE_KIND" == "true" ]]; then
    YAML=${YAML//imagePullPolicy: Always/imagePullPolicy: IfNotPresent}
fi

echo "Deploying conformance runner..."
echo "${YAML}" | kubectl ${KUBECONFIG_ARG} apply -f -

# A private registry needs a pull secret; KIND side-loads the runner image and
# needs none. When the local context carries registry credentials, mint it,
# attach it to the runner's ServiceAccount and restart so the pod is recreated
# inheriting it.
pull_secret=$(ensure_pull_secret "${NAMESPACE}")
if [ -n "${pull_secret}" ]; then
    kubectl ${KUBECONFIG_ARG} -n "${NAMESPACE}" patch serviceaccount conformance-sa \
        --type merge -p "{\"imagePullSecrets\":[{\"name\":\"${pull_secret}\"}]}"
    kubectl ${KUBECONFIG_ARG} -n "${NAMESPACE}" rollout restart deploy/conformance
fi

echo "Waiting for the conformance pod to be ready..."
kubectl ${KUBECONFIG_ARG} -n "${NAMESPACE}" rollout status deploy/conformance --timeout=120s

POD=$(kubectl ${KUBECONFIG_ARG} -n "${NAMESPACE}" get pod -l app=conformance -o jsonpath='{.items[0].metadata.name}')
echo "--- Running conformance (${CONFORMANCE_SCENARIOS}) in pod ${POD} ---"
kubectl ${KUBECONFIG_ARG} -n "${NAMESPACE}" exec "${POD}" -- ./run-conformance.sh "${CONFORMANCE_SCENARIOS}"
