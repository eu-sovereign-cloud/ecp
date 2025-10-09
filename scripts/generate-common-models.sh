#!/bin/bash

GREEN="\033[1;32m"
RESET="\033[0m"

out_dir="spec/spec"
sdk_dir="./internal/go-sdk/pkg/spec/schema"

set -e

if [ "$#" -eq 0 ]; then
    echo "Usage: $0 <model1> [<model2> ...]"
    echo "Available common models: 'errors', 'resource'"
    exit 1
fi

echo "Creating output directory at ${out_dir} if it doesn't exist..."
mkdir -p ${out_dir}

for model in "$@"; do
    model_path="apis/generated/types/${model}/zz_generated_${model}.go"
    echo "Copying Go code from SDK for ${model} models..."
    mkdir -p "apis/generated/types/${model}"
    cp "${sdk_dir}/${model}.go" "${model_path}"

    echo -e "${GREEN}✅ Go code copied successfully.${RESET}"
    echo "Replacing time package types with metav1 package time types and fixing map types..."

    sed -i "s/^package .*/package ${model}/" "${model_path}"
    sed -i 's/time\.Time/metav1.Time/g' "${model_path}"
    sed -i 's/map\[string\]interface{}/\*map[string]string/g' "${model_path}"
    sed -i 's|.*"time".*|	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"|' "${model_path}"

    echo -e "${GREEN}✅ Type replacements completed successfully.${RESET}"

    echo "Add +kubebuilder:object:root=true annotations to common package..."
    sed -i "/^package ${model}/a\// +kubebuilder:object:generate=true" "${model_path}"
    echo -e "${GREEN}✅ Annotations added successfully to ${model_path} .${RESET}"

    echo "Generating DeepCopy methods for common models..."
    ${GO_TOOL} -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object paths="./apis/generated/types/${model}/"

    echo -e "${GREEN}✅ DeepCopy methods generated successfully.${RESET}"
    echo -e "${GREEN}All tasks completed successfully for ${model} models!${RESET}\n"
done

echo -e "${GREEN}🎉 All specified models processed successfully!${RESET}"
