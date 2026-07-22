#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
check_component_arg "$1"
source_config

setup_kube_vars
setup_registry_vars "$1"

DEPLOY_DIR="${SCRIPT_DIR}/../deploy/${COMPONENT}"
CRDS_DIR="${SCRIPT_DIR}/../../../chart/crd"

echo "Applying CRDs from ${CRDS_DIR}..."
find "${CRDS_DIR}" -type f -name "*.yaml" -exec cat {} + | kubectl ${KUBECONFIG_ARG} apply -f -

echo "Deploying ${COMPONENT} with image ${IMAGE_NAME}..."

# Build the YAML stream from kustomize
YAML_STREAM=$(kubectl kustomize "${DEPLOY_DIR}")

# Replace the image placeholder first
YAML_STREAM=$(echo "${YAML_STREAM}" | sed "s|##IMAGE_NAME##|${IMAGE_NAME}|g")

# Replace image pull policy when runnin on kIND
if [[ -n "$USE_KIND" && "$USE_KIND" == "true" ]]; then
    YAML_STREAM=$(echo "${YAML_STREAM}" | sed "s|imagePullPolicy: Always|imagePullPolicy: IfNotPresent|g")
fi

# Retarget the component namespace (default e2e-ecp). Anchored to end-of-line so
# only namespace values are rewritten, not the e2e-ecp-conformance resource name.
SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-e2e-ecp}"
if [ "$SYSTEM_NAMESPACE" != "e2e-ecp" ]; then
    echo "Retargeting namespace to ${SYSTEM_NAMESPACE}"
    YAML_STREAM=$(echo "${YAML_STREAM}" | sed -E "s/e2e-ecp[[:space:]]*$/${SYSTEM_NAMESPACE}/")
fi

# Retarget the fixture tenant (default test-tenant). Its CRs live in the ECP tenant
# namespace hex(sha3-224(tenant)) (framework/backend/kubernetes/adapter.go
# ComputeNamespace), so rewrite both the tenant string and its hashed namespace.
if [ "$COMPONENT" == "test-data" ]; then
    E2E_TENANT="${E2E_TENANT:-test-tenant}"
    if [ "$E2E_TENANT" != "test-tenant" ]; then
        echo "Retargeting tenant to ${E2E_TENANT}"
        old_ns=$(printf %s "test-tenant" | openssl dgst -sha3-224 | awk '{print $NF}')
        new_ns=$(printf %s "$E2E_TENANT"  | openssl dgst -sha3-224 | awk '{print $NF}')
        YAML_STREAM=$(echo "${YAML_STREAM}" | sed -e "s/${old_ns}/${new_ns}/g" -e "s/test-tenant/${E2E_TENANT}/g")
    fi
fi

# Both gateways pick their authentication plugin at deploy time: dummy (default)
# or jwt. The suites read the same AUTH_PLUGIN to mint matching tokens, so deploy
# and test with the same value (see test/Makefile).
if [[ "$COMPONENT" == gateway-* ]]; then
    echo "Deploying ${COMPONENT} with auth plugin: ${AUTH_PLUGIN:=dummy}"
    YAML_STREAM=$(echo "${YAML_STREAM}" | sed "s|##AUTH_PLUGIN##|${AUTH_PLUGIN}|g")
fi

# If the component is the delegator, handle the plugin type.
# PLUGIN_TYPE may be overridden via the environment; otherwise default to
# aruba, except on KIND where the dummy plugin is the default.
if [ "$COMPONENT" == "delegator" ]; then
    if [ -z "$PLUGIN_TYPE" ]; then
        PLUGIN_TYPE="aruba" # Default to aruba
        if [[ -n "$USE_KIND" && "$USE_KIND" == "true" ]]; then
            PLUGIN_TYPE="dummy"
        fi
    fi
    echo "Deploying delegator with plugin: ${PLUGIN_TYPE}"
    YAML_STREAM=$(echo "${YAML_STREAM}" | sed "s|##PLUGIN_TYPE##|${PLUGIN_TYPE}|g")
fi

# Apply the processed YAML stream
echo "${YAML_STREAM}" | kubectl ${KUBECONFIG_ARG} apply -f -

# Wait for the rollout, otherwise a suite starting right after can port-forward to
# the terminating pod, which still serves the previous config (e.g. the previous
# auth plugin) and 401s every valid token. Components without a Deployment
# (test-data) skip this.
if kubectl ${KUBECONFIG_ARG} -n "${SYSTEM_NAMESPACE}" get deployment "${COMPONENT}-depl" >/dev/null 2>&1; then
    kubectl ${KUBECONFIG_ARG} -n "${SYSTEM_NAMESPACE}" rollout status "deployment/${COMPONENT}-depl" --timeout=180s
fi

echo "Deployment of ${COMPONENT} complete."

