#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
check_component_arg "$1"
source_config

setup_kube_vars
setup_registry_vars "$1"

DEPLOY_DIR="${SCRIPT_DIR}/../deploy/${COMPONENT}"
CRDS_DIR="${SCRIPT_DIR}/../../../chart/crds"

# Retarget the component namespace (default e2e-ecp).
SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-e2e-ecp}"

echo "Applying CRDs from ${CRDS_DIR}..."
find "${CRDS_DIR}" -type f -name "*.yaml" -exec cat {} + | kubectl ${KUBECONFIG_ARG} apply -f -

echo "Deploying ${COMPONENT} with image ${IMAGE_NAME}..."

# --- Chart-deployed components -----------------------------------------------
# Deployed from the charts this repository ships, so the test stack and a real
# `helm install` exercise the same templates: a broken template fails a test run
# instead of a user's first install. Everything the manifests used to hardcode
# (image, pull policy, auth plugin, namespace) is a value.
if setup_chart_vars "${COMPONENT}"; then
    kubectl ${KUBECONFIG_ARG} create namespace "${SYSTEM_NAMESPACE}" --dry-run=client -o yaml |
        kubectl ${KUBECONFIG_ARG} apply -f -

    # IMAGE_NAME is repository:tag; the charts take the halves separately.
    HELM_ARGS=(
        --namespace "${SYSTEM_NAMESPACE}"
        --set "${IMAGE_VALUE_PATH}.repository=${IMAGE_NAME%:*}"
        --set "${IMAGE_VALUE_PATH}.tag=${IMAGE_NAME##*:}"
    )

    # A private registry needs a pull secret; KIND and public GHCR do not. When
    # the local context carries registry credentials, mint it and point the chart
    # at it — both charts take imagePullSecrets by name.
    pull_secret=$(ensure_pull_secret "${SYSTEM_NAMESPACE}")
    if [ -n "${pull_secret}" ]; then
        HELM_ARGS+=(--set "imagePullSecrets[0].name=${pull_secret}")
    fi

    # Locally built images are side-loaded into KIND and never pullable.
    if [[ "$USE_KIND" == "true" ]]; then
        HELM_ARGS+=(--set "${IMAGE_VALUE_PATH}.pullPolicy=IfNotPresent")
    else
        HELM_ARGS+=(--set "${IMAGE_VALUE_PATH}.pullPolicy=Always")
    fi

    # Both gateways pick their authentication plugin at deploy time: dummy
    # (default) or jwt. The suites read the same AUTH_PLUGIN to mint matching
    # tokens, so deploy and test with the same value (see test/Makefile).
    if [[ "$COMPONENT" == gateway-* ]]; then
        echo "Deploying ${COMPONENT} with auth plugin: ${AUTH_PLUGIN:=dummy}"
        HELM_ARGS+=(
            --values "${SCRIPT_DIR}/../deploy/gateway-values.yaml"
            --set "auth.plugin=${AUTH_PLUGIN}"
        )
        # The RBAC checker to benchmark: the report workflow deploys the same
        # gateway twice, cached and direct, and diffs the two metric snapshots
        # (see README, "Benchmarking the Auth Middleware").
        if [ -n "${AUTHZ_IMPL:-}" ]; then
            HELM_ARGS+=(--set "auth.authz.impl=${AUTHZ_IMPL}")
        fi
    fi

    # The delegator's CSP plugin set. PLUGIN_TYPE may be overridden via the
    # environment; otherwise default to aruba, except on KIND where the dummy
    # plugin is the only self-contained one.
    if [ "$COMPONENT" == "delegator" ]; then
        if [ -z "$PLUGIN_TYPE" ]; then
            PLUGIN_TYPE="aruba"
            if [[ "$USE_KIND" == "true" ]]; then
                PLUGIN_TYPE="dummy"
            fi
        fi
        echo "Deploying delegator with plugin: ${PLUGIN_TYPE}"
        HELM_ARGS+=(--set "plugin=${PLUGIN_TYPE}")
    fi

    # Per-component values, then an optional overlay (DEPLOY_VALUES) layered last
    # so it wins — e.g. the multicluster topology exposes the regional gateway on
    # a NodePort (see the kind-multicluster-stack target).
    VALUES_ARGS=(--values "${DEPLOY_DIR}/values.yaml")
    if [ -n "${DEPLOY_VALUES:-}" ]; then
        VALUES_ARGS+=(--values "${DEPLOY_VALUES}")
    fi

    # --wait replaces the explicit rollout wait: a suite starting right after
    # would otherwise port-forward to the terminating pod, which still serves
    # the previous config (e.g. the previous auth plugin) and 401s every valid
    # token.
    helm ${HELM_KUBECONFIG_ARG} upgrade --install "${HELM_RELEASE}" "${CHART_DIR}" \
        "${HELM_ARGS[@]}" "${VALUES_ARGS[@]}" --wait --timeout 3m

    echo "Deployment of ${COMPONENT} complete."
    exit 0
fi

# --- Kustomize-deployed components -------------------------------------------
# Build the YAML stream from kustomize
YAML_STREAM=$(kubectl kustomize "${DEPLOY_DIR}")

# Replace the image placeholder first
YAML_STREAM=$(echo "${YAML_STREAM}" | sed "s|##IMAGE_NAME##|${IMAGE_NAME}|g")

# Replace image pull policy when runnin on kIND
if [[ -n "$USE_KIND" && "$USE_KIND" == "true" ]]; then
    YAML_STREAM=$(echo "${YAML_STREAM}" | sed "s|imagePullPolicy: Always|imagePullPolicy: IfNotPresent|g")
fi

# Anchored to end-of-line so only namespace values are rewritten, not the
# e2e-ecp-conformance resource name.
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

# Apply the processed YAML stream
echo "${YAML_STREAM}" | kubectl ${KUBECONFIG_ARG} apply -f -

# Wait for the rollout, otherwise a suite starting right after can port-forward to
# the terminating pod, which still serves the previous config. Components without
# a Deployment (test-data) skip this.
if kubectl ${KUBECONFIG_ARG} -n "${SYSTEM_NAMESPACE}" get deployment "${COMPONENT}-depl" >/dev/null 2>&1; then
    kubectl ${KUBECONFIG_ARG} -n "${SYSTEM_NAMESPACE}" rollout status "deployment/${COMPONENT}-depl" --timeout=180s
fi

echo "Deployment of ${COMPONENT} complete."
