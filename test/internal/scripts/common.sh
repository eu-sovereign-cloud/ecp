#!/bin/bash

# This script is meant to be sourced.
# It provides common functions and variables for other scripts.

# Fail on any error
set -eo pipefail

# setup_env initializes common directory variables.
setup_env() {
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
    CONTEXT_DIR="${SCRIPT_DIR}/../context"
}

# check_component_arg validates that a component has been passed as an argument.
check_component_arg() {
    if [ -z "$1" ]; then
        echo "Usage: $0 <component>" >&2
        exit 1
    fi
    COMPONENT=$1
}

# setup_registry_vars sets the IMAGE_NAME environment variable based on context.
# Usage: setup_registry_vars <component>
setup_registry_vars() {
    local component=${1:?component is required}

    local version=${VERSION:-"latest"}
    local img_base=${IMG:-"e2e-ecp-${component}"}
    local registry_url
    local registry_project

    if [[ -n "$USE_KIND" && "$USE_KIND" == "true" ]]; then
        registry_url="localhost"
        registry_project="" # No project for local kind images
    else
        # Use environment's settings, with a fallback for registry_url
        registry_url=${REGISTRY_URL:-"localhost"}
        # Clean the project name
        registry_project=${REGISTRY_PROJECT}
    fi

    local image_name="${img_base}:${version}"
    if [ -n "$registry_project" ]; then
        image_name="${registry_project}/${image_name}"
    fi
    if [ -n "$registry_url" ]; then
        image_name="${registry_url}/${image_name}"
    fi

    export IMAGE_NAME="${image_name}"
}

# Service names the gateway chart renders, "<release>-<component>". Anything
# dialling a gateway by DNS rather than by port-forward needs them: the
# conformance runner, the benchmark scraper, and test-data/regions.yaml, which
# publishes them in the region catalog for clients to follow. Keep the YAML in
# step with these.
GATEWAY_GLOBAL_SVC="ecp-global-gateway-global"
GATEWAY_REGIONAL_SVC="ecp-regional-gateway-regional"

# setup_chart_vars maps a component to the Helm chart that deploys it, setting
# CHART_DIR, HELM_RELEASE and IMAGE_VALUE_PATH (the values key holding the image
# coordinates, which differs between the two charts). Returns 1 for components
# that are not chart-deployed — test-data is fixture CRs and conformance is the
# secatest runner, neither of which is something anyone installs — so callers
# can branch on it:
#
#     if setup_chart_vars "$COMPONENT"; then helm ...; else kubectl kustomize ...; fi
#
# Two releases share the gateway chart, each disabling the other gateway, so a
# component can still be deployed on its own. Their names must contain the chart
# name ("ecp") for the fullname helper to use them verbatim.
setup_chart_vars() {
    local component=${1:?component is required}
    local root="${SCRIPT_DIR}/../../.."

    case "${component}" in
        gateway-global)
            CHART_DIR="${root}/chart"; HELM_RELEASE="ecp-global"; IMAGE_VALUE_PATH="gatewayGlobal.image" ;;
        gateway-regional)
            CHART_DIR="${root}/chart"; HELM_RELEASE="ecp-regional"; IMAGE_VALUE_PATH="gatewayRegional.image" ;;
        *)
            return 1 ;;
    esac
}

# setup_kube_vars sets the KUBECONFIG_ARG, KUBECONFIG, and CLUSTER_NAME environment variables.
setup_kube_vars() {
    # 1. Handle KIND case
    if [[ -n "$USE_KIND" && "$USE_KIND" == "true" ]]; then
        unset KUBECONFIG
        # KIND_CLUSTER selects which KIND cluster to act on; the single-cluster
        # flows leave it unset and get e2e-cluster. The multicluster flow sets it
        # per invocation to drive two clusters from the same scripts.
        export CLUSTER_NAME="${KIND_CLUSTER:-e2e-cluster}"
        # Only address the cluster by context when one was explicitly selected.
        # `kind create cluster` points the ambient current-context at whichever
        # cluster it made last, so with two clusters ambient targeting silently
        # picks one — but the single-cluster flows have always relied on ambient
        # targeting, so leave their behaviour byte-identical.
        if [ -n "${KIND_CLUSTER}" ]; then
            KUBECONFIG_ARG="--context kind-${CLUSTER_NAME}"
        else
            KUBECONFIG_ARG=""
        fi
        return
    fi

    # 2. Handle context file case
    local context_kubeconfig="${CONTEXT_DIR}/kubeconfig.yaml"
    if [ -f "${context_kubeconfig}" ]; then
        export KUBECONFIG="${context_kubeconfig}"
        KUBECONFIG_ARG="--kubeconfig ${context_kubeconfig}"
        local current_context
        current_context=$(kubectl config current-context --kubeconfig "${context_kubeconfig}")
        export CLUSTER_NAME
        CLUSTER_NAME=$(kubectl config view --kubeconfig "${context_kubeconfig}" -o jsonpath="{.contexts[?(@.name==\"${current_context}\")].context.cluster}")
        return
    fi

    # 3. Default: honour an exported KUBECONFIG (custom cluster) or ~/.kube/config.
    KUBECONFIG_ARG=""
    local current_context
    current_context=$(kubectl config current-context)
    export CLUSTER_NAME
    CLUSTER_NAME=$(kubectl config view -o jsonpath="{.contexts[?(@.name==\"${current_context}\")].context.cluster}")
}

# source_config sources the config.env file if it exists.
source_config() {
    if [ -f "${CONTEXT_DIR}/config.env" ]; then
        source "${CONTEXT_DIR}/config.env"
    fi
}