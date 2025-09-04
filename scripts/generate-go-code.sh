#!/bin/bash

BLUE="\033[1;34m"
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
RESET="\033[0m"

set -e

if [ "$#" -lt 4 ]; then
    echo "Usage: $0 <output-file> <spec-file> <crd-output-dir> <common-model1> [<common-model2> ...]"
    echo "Available common models: 'errors', 'resource'"
    exit 1
fi

OUTPUT_FILE=$1
SPEC_FILE=$2
CRD_OUTPUT_DIR=$3
COMMON_MODELS=( ${@:4} )

excluded_schemas=""
for model in "${COMMON_MODELS[@]}"; do
    schemas=$(cat "./apis/common/${model}/zz_generated_${model}.go" | grep --perl-regexp '^(?!\/\/)type' | tr -s ' ' | cut -d ' ' -f "2" | tr '\n' ',')
    schemas=${schemas%,}
    if [ -n "$schemas" ]; then
        if [ -n "$excluded_schemas" ]; then
            excluded_schemas="${excluded_schemas},${schemas}"
        else
            excluded_schemas="${schemas}"
        fi
    fi
done

echo "Generating Go code from $SPEC_FILE to $OUTPUT_FILE"

${GO_TOOL} -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --generate=types \
    --exclude-schemas="${excluded_schemas}" \
    -o "${OUTPUT_FILE}" -package v1 "${SPEC_FILE}"

echo -e "${GREEN}✅ Go code generated successfully.${RESET}"

for model in "${COMMON_MODELS[@]}"; do
    schemas=$(cat "./apis/common/${model}/zz_generated_${model}.go" | grep --perl-regexp '^(?!\/\/)type' | tr -s ' ' | cut -d ' ' -f "2" | tr '\n' ',')
    schemas=${schemas%,}
    IFS="," schemas_array=( ${schemas} )

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
            sed -i --regexp-extended "s/package v1/package v1\n\nimport \(\n\t${model} \"github\.com\/eu\-sovereign\-cloud\/ecp\/apis\/common\/${model}\"\n)/g" ${OUTPUT_FILE}
        else
            sed -i --regexp-extended "s/import \(/import \(\n\t${model} \"github\.com\/eu\-sovereign\-cloud\/ecp\/apis\/common\/${model}\"/g" ${OUTPUT_FILE}
        fi
    fi

    echo -e "${GREEN}✅ Post-processing for common/${model} completed successfully.${RESET}\n"
done

echo "Replacing time package types with k8s metav1 package time types and fixing map types in the generated file..."

# Replace time.Time with metav1.Time and fix map types in the generated file
# Add import for metav1 if needed
sed -i 's/time\.Time/metav1.Time/g' ${OUTPUT_FILE}
sed -i 's/map\[string\]interface{}/\*map[string]string/g' ${OUTPUT_FILE}
sed -i 's|.*"time".*|	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"|' ${OUTPUT_FILE}

echo -e "${GREEN}✅ Replacements for time package types and map types completed successfully.${RESET}"

gofmt -w ${OUTPUT_FILE}

echo "Preparing CRD output directory..."
# Clean up and create the output directory
rm -rf "$CRD_OUTPUT_DIR"
mkdir -p "$CRD_OUTPUT_DIR"
echo "CRD output directory is ready at $CRD_OUTPUT_DIR."