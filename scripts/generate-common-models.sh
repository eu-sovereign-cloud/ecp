#!/bin/bash

BLUE="\033[1;34m"
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
RESET="\033[0m"

template_file="scripts/common-template/dummy.yaml.tpl"
out_dir="spec/spec"
schemas_dir="./schemas"

set -e

if [ "$#" -eq 0 ]; then
    echo "Usage: $0 <model1> [<model2> ...]"
    echo "Available common models: 'errors', 'resource'"
    exit 1
fi

gomplate --version >/dev/null || {
    echo -e "${YELLOW}⚠️  gomplate command not found or not executable. Please install gomplate.\n    For instructions visit ${BLUE}https://docs.gomplate.ca/installing/${RESET}"
    exit 1
}

echo "Creating output directory at ${out_dir} if it doesn't exist..."
mkdir -p ${out_dir}

for model in "$@"; do
    echo "Generating OpenAPI dummy spec files for ${model} models..."
    echo '{"dir": "'${schemas_dir}'", "file": "'"${model}"'.yaml"}' | gomplate -f ${template_file} -o "${out_dir}/dummy-${model}-spec.yaml" -d data="spec/spec/schemas/${model}.yaml" -d path=stdin:path.json

    echo -e "${GREEN}✅ Dummy OpenAPI spec files generated successfully.${RESET}"
    echo "Bundling dummy OpenAPI spec files into a single file with ${model} models..."

    npx @redocly/cli bundle --remove-unused-components "${out_dir}/dummy-${model}-spec.yaml" -o "${out_dir}/${model}-bundled.yaml"

    echo -e "${GREEN}✅ Bundling completed successfully.${RESET}"

    echo "Cleaning up dummy spec files..."
    rm ${out_dir}/dummy-*-spec.yaml

    excluded_schemas=$(cat "${out_dir}/${model}-bundled.yaml" | grep "Excluded:" | sed 's/://g' | tr '\n' ',' | tr -s ' ' | sed 's/ //g')
    excluded_schemas=${excluded_schemas%,}

    echo "Generating Go code from bundled OpenAPI spec files for ${model} models..."

    mkdir -p "apis/generated/types/${model}"

    ${GO_TOOL} -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --generate=types \
        --exclude-schemas="${excluded_schemas}" \
        -o "apis/generated/types/${model}/zz_generated_${model}.go" -package "${model}" "${out_dir}/${model}-bundled.yaml"

    echo -e "${GREEN}✅ Go code generated successfully.${RESET}"
    echo "Replacing time package types with metav1 package time types and fixing map types..."

    sed -i 's/time\.Time/metav1.Time/g' "apis/generated/types/${model}/zz_generated_${model}.go"
    sed -i 's/map\[string\]interface{}/\*map[string]string/g' "apis/generated/types/${model}/zz_generated_${model}.go"
    sed -i 's|.*"time".*|	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"|' "apis/generated/types/${model}/zz_generated_${model}.go"

    echo -e "${GREEN}✅ Type replacements completed successfully.${RESET}"

    echo "Add +kubebuilder:object:root=true annotations to common packages..."
    sed -i "/^package ${model}/a\// +kubebuilder:object:generate=true" "apis/generated/types/${model}/zz_generated_${model}.go"
    echo -e "${GREEN}✅ Annotations added successfully.${RESET}"

    echo "Generating DeepCopy methods for common models..."
    ${GO_TOOL} -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object paths="./apis/generated/types/${model}/"

    echo -e "${GREEN}✅ DeepCopy methods generated successfully.${RESET}"
    echo -e "${GREEN}All tasks completed successfully for ${model} models!${RESET}\n"
done

echo -e "${GREEN}🎉 All specified models processed successfully!${RESET}"
