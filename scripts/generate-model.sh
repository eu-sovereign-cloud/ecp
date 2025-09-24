#!/bin/bash

BLUE="\033[1;34m"
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
RESET="\033[0m"

set -e

if [ "$1" = "help" ]; then
    echo "Usage: $0 <spec-path> <common-model1> [<common-model2> ...]"
    echo "Available common models: 'errors', 'resource'"
    echo "Output directory will be './apis/generated/types/<api-name>/<api-version>' (it will be created if it doesn't exist)."
    exit 0
fi

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <spec-path> <common-model1> [<common-model2> ...]"
    echo "Available common models: 'errors', 'resource'"
    exit 1
fi

SPEC_PATH=$1
COMMON_MODELS=( ${@:2} )

API_NAME=$(basename "$SPEC_PATH" | cut -d '.' -f 2)
API_VERSION=$(basename "$SPEC_PATH" | cut -d '.' -f 3)
OUTPUT_DIR="./apis/generated/types/${API_NAME}/${API_VERSION}"
OUTPUT_FILE="${OUTPUT_DIR}/zz_generated_${API_NAME}.go"

excluded_schemas=""
for model in "${COMMON_MODELS[@]}"; do
    schemas=$(cat "./apis/generated/types/${model}/zz_generated_${model}.go" | grep --perl-regexp '^(?!\/\/)type' | tr -s ' ' | cut -d ' ' -f "2" | tr '\n' ',')
    schemas=${schemas%,}
    if [ -n "$schemas" ]; then
        if [ -n "$excluded_schemas" ]; then
            excluded_schemas="${excluded_schemas},${schemas}"
        else
            excluded_schemas="${schemas}"
        fi
    fi
done

echo "Generating Go code from $SPEC_PATH to $OUTPUT_FILE"

mkdir -p "${OUTPUT_DIR}"
${GO_TOOL} -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --generate=types \
    --exclude-schemas="${excluded_schemas}" \
    -o "${OUTPUT_FILE}" -package "${API_VERSION}" "${SPEC_PATH}"

echo -e "${GREEN}✅ Go code generated successfully.${RESET}\n"

for model in "${COMMON_MODELS[@]}"; do
    schemas=$(grep --perl-regexp '^(?!\/\/)type' "./apis/generated/types/${model}/zz_generated_${model}.go" | tr -s ' ' | cut -d ' ' -f "2" | tr '\n' ',')
    schemas=${schemas%,}
    schemas_array=( $(echo "${schemas}" | tr ',' ' ') )

    requires_import=false

    echo "Post-processing the generated Go code to replace excluded schemas with references to common/${model} package..."
    for schema in "${schemas_array[@]}"; do
        # Remove potential duplicate type definitions caused by oapi-codegen not respecting --exclude-schemas fully
        sed -i --regexp-extended "s/(type|\/\/) ${schema}.*//g" ${OUTPUT_FILE}
        if [ $(grep -c "${schema}" ${OUTPUT_FILE}) -gt 0 ]; then
            requires_import=true
        fi

        sed -i "/^\/\//! s/ ${schema}/ ${model}.${schema}/g" ${OUTPUT_FILE}
        sed -i "/^\/\//! s/\*${schema}/*${model}.${schema}/g" ${OUTPUT_FILE}
        sed -i "/^\/\//! s/\]${schema}/]${model}.${schema}/g" ${OUTPUT_FILE}
    done

    if [ "$requires_import" = true ]; then
        if [ $(grep -c "import" ${OUTPUT_FILE}) -eq 0 ]; then
            sed -i --regexp-extended "s/package ${API_VERSION}/package ${API_VERSION}\n\nimport \(\n\t${model} \"github\.com\/eu\-sovereign\-cloud\/ecp\/apis\/generated\/types\/${model}\"\n)/g" ${OUTPUT_FILE}
        else
            sed -i --regexp-extended "s/import \(/import \(\n\t${model} \"github\.com\/eu\-sovereign\-cloud\/ecp\/apis\/generated\/types\/${model}\"/g" ${OUTPUT_FILE}
        fi
    fi

    echo -e "${GREEN}✅ Post-processing for types/${model} completed successfully.${RESET}\n"
done

echo "Replacing time package types with k8s metav1 package time types and fixing map types in the generated file..."

# Replace time.Time with metav1.Time and fix map types in the generated file
# Add import for metav1 if needed
sed -i 's/time\.Time/metav1.Time/g' ${OUTPUT_FILE}
sed -i 's/map\[string\]interface{}/\*map[string]string/g' ${OUTPUT_FILE}
sed -i 's|.*"time".*|	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"|' ${OUTPUT_FILE}

echo -e "${GREEN}✅ Replacements for time package types and map types completed successfully.${RESET}\n"

gofmt -w ${OUTPUT_FILE}

echo "Add +kubebuilder:object:root=true annotation to ${OUTPUT_FILE}..."
sed -i "/^package ${API_VERSION}/a\// +kubebuilder:object:generate=true" "${OUTPUT_FILE}"
echo -e "${GREEN}✅ Annotations added successfully.${RESET}\n"

echo "Generating DeepCopy methods for ${API_NAME} models..."
${GO_TOOL} -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object paths="${OUTPUT_DIR}"

echo -e "${GREEN}✅ DeepCopy methods generated successfully.${RESET}"
echo -e "${GREEN}🎉 All tasks completed successfully for ${API_NAME} ${API_VERSION} models!${RESET}\n"
