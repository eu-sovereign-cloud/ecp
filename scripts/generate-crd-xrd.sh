#!/bin/bash

set -e

BLUE="\033[1;34m"
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
RESET="\033[0m"

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <API>"
    exit 1
fi

API=$1
CRD_TYPES_DIR="./apis/${API}/crds/v1"
XRD_TYPES_DIR="./apis/${API}/xrds/v1"
CRD_OUTPUT_DIR="./apis/generated/crds/${API}"
XRD_OUTPUT_DIR="./apis/generated/xrds/${API}"

if [ -d "${CRD_TYPES_DIR}" ]; then
    echo "Generating CRDs for ${API}..."
    ${GO_TOOL} -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=".github/boilerplate.go.txt" paths="${CRD_TYPES_DIR}/..."
    ${GO_TOOL} -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen crd paths="${CRD_TYPES_DIR}/..." output:crd:artifacts:config="${CRD_OUTPUT_DIR}"
    echo -e "${GREEN}✅ CRDs for ${API} generated in ${CRD_OUTPUT_DIR}.\n${RESET}"
else
    echo -e "${YELLOW}Skipping CRD generation for ${API}, no CRD types found.\n${RESET}"
fi

if [ -d "${XRD_TYPES_DIR}" ]; then
    echo "Generating XRDs for ${API}..."
    ${GO_TOOL} -mod=mod github.com/mproffitt/crossbuilder/cmd/xrd-gen object:headerFile=".github/boilerplate.go.txt" paths="${XRD_TYPES_DIR}/..."
    ${GO_TOOL} -mod=mod github.com/mproffitt/crossbuilder/cmd/xrd-gen xrd paths="${XRD_TYPES_DIR}/..." output:xrd:artifacts:config="${XRD_OUTPUT_DIR}"
    echo -e "${GREEN}✅ XRDs for ${API} generated in ${XRD_OUTPUT_DIR}.\n${RESET}"
else
    echo -e "${YELLOW}Skipping XRD generation for ${API}, no XRD types found.\n${RESET}"
fi
