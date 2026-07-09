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

# setup_kube_vars sets the KUBECONFIG_ARG, KUBECONFIG, and CLUSTER_NAME environment variables.
setup_kube_vars() {
    # 1. Handle KIND case
    if [[ -n "$USE_KIND" && "$USE_KIND" == "true" ]]; then
        unset KUBECONFIG
        KUBECONFIG_ARG=""
        export CLUSTER_NAME="e2e-cluster" # Hardcoded for KIND
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

    # 3. Handle default case
    unset KUBECONFIG
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