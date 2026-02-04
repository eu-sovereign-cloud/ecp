#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
check_component_arg "$1"
source_config

setup_kube_vars
setup_registry_vars "$1"

DEPLOY_DIR="${SCRIPT_DIR}/../deploy/${COMPONENT}"
CRDS_DIR="${SCRIPT_DIR}/../../../api/generated/crds"

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

# If the component is the delegator, handle the plugin type
if [ "$COMPONENT" == "delegator" ]; then
    PLUGIN_TYPE="aruba" # Default to aruba
    if [[ -n "$USE_KIND" && "$USE_KIND" == "true" ]]; then
        PLUGIN_TYPE="dummy"
    fi
    echo "Deploying delegator with plugin: ${PLUGIN_TYPE}"
    YAML_STREAM=$(echo "${YAML_STREAM}" | sed "s|##PLUGIN_TYPE##|${PLUGIN_TYPE}|g")
fi

# Apply the processed YAML stream
echo "${YAML_STREAM}" | kubectl ${KUBECONFIG_ARG} apply -f -

echo "Deployment of ${COMPONENT} complete."

