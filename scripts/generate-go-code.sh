#!/bin/bash

BLUE="\033[1;34m"
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
RESET="\033[0m"

set -e

if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <output-file> <spec-file> <crd-output-dir>"
    exit 1
fi

OUTPUT_FILE=$1
SPEC_FILE=$2
CRD_OUTPUT_DIR=$3

resources_schemas=$(cat ./apis/common/resources/zz_generated_resources.go | grep --perl-regexp '^(?!\/\/)type' | tr -s ' ' | cut -d ' ' -f "2" | tr '\n' ',')
resources_schemas=${resources_schemas%,}

errors_schemas=$(cat ./apis/common/errors/zz_generated_errors.go | grep --perl-regexp '^(?!\/\/)type' | tr -s ' ' | cut -d ' ' -f "2" | tr '\n' ',')
errors_schemas=${errors_schemas%,}

excluded_schemas=$(echo "${resources_schemas},${errors_schemas}")

echo "Generating Go code from $SPEC_FILE to $OUTPUT_FILE"

${GO_TOOL} -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --generate=types \
    --exclude-schemas=${excluded_schemas} \
    -o ${OUTPUT_FILE} -package v1 ${SPEC_FILE}

echo -e "${GREEN}✅ Go code generated successfully.${RESET}"

requires_common_resources_import=false
requires_common_errors_import=false

IFS="," resources_array=( ${resources_schemas} )
IFS="," errors_array=( ${errors_schemas} )

echo "Post-processing the generated Go code to replace excluded schemas with references to common/resources package..."

# Post-process the generated code to replace excluded schemas with references to common/resources package
for schema in "${resources_array[@]}"; do
    sed -i --regexp-extended "s/(type|\/\/) ${schema}.*//g" ${OUTPUT_FILE}
    if [ $(grep -c "${schema}" ${OUTPUT_FILE}) -gt 0 ]; then
        requires_common_resources_import=true
    fi

    # Replace references to excluded schemas with types defined in common/resources package
    # Ignore appearances in comments
    sed -i "/^\/\//! s/ ${schema}/ resources.${schema}/g" ${OUTPUT_FILE}
    sed -i "/^\/\//! s/\*${schema}/*resources.${schema}/g" ${OUTPUT_FILE}
    sed -i "/^\/\//! s/\]${schema}/]resources.${schema}/g" ${OUTPUT_FILE}
done

if [ "$requires_common_resources_import" = true ]; then
    if [ $(grep -c "import" ${OUTPUT_FILE}) -eq 0 ]; then
        sed -i --regexp-extended "s/package v1/package v1\n\nimport \(\n\tresources \"github\.com\/eu\-sovereign\-cloud\/ecp\/apis\/common\/resources\"\n)/g" ${OUTPUT_FILE}
    else
        sed -i --regexp-extended "s/import \(/import \(\n\tresources \"github\.com\/eu\-sovereign\-cloud\/ecp\/apis\/common\/resources\"/g" ${OUTPUT_FILE}
    fi
fi

echo -e "${GREEN}✅ Post-processing for common/resources completed successfully.${RESET}"
echo "Post-processing the generated Go code to replace excluded schemas with references to common/errors package..."

# Post-process the generated code to replace excluded schemas with references to common/errors package
for schema in "${errors_array[@]}"; do
    sed -i --regexp-extended "s/(type|\/\/) ${schema}.*//g" ${OUTPUT_FILE}
    if [ $(grep -c "${schema}" ${OUTPUT_FILE}) -gt 0 ]; then
        requires_common_errors_import=true
    fi

    # Replace references to excluded schemas with types defined in common/errors package
    # Ignore appearances in comments
    sed -i "/^\/\//! s/ ${schema}/ errors.${schema}/g" ${OUTPUT_FILE}
    sed -i "/^\/\//! s/\*${schema}/*errors.${schema}/g" ${OUTPUT_FILE}
    sed -i "/^\/\//! s/\]${schema}/]errors.${schema}/g" ${OUTPUT_FILE}
done

if [ "$requires_common_errors_import" = true ]; then
    if [ $(grep -c "import" ${OUTPUT_FILE}) -eq 0 ]; then
        sed -i --regexp-extended "s/package v1/package v1\n\nimport \(\n\terrors \"github\.com\/eu\-sovereign\-cloud\/ecp\/apis\/common\/errors\"\n)/g" ${OUTPUT_FILE}
    else
        sed -i --regexp-extended "s/import \(/import \(\n\terrors \"github\.com\/eu\-sovereign\-cloud\/ecp\/apis\/common\/errors\"/g" ${OUTPUT_FILE}
    fi
fi

echo -e "${GREEN}✅ Post-processing for common/errors completed successfully.${RESET}"
echo "Replacing time package types with k8s metav1 package time types and fixing map types in the generated file..."

# Replace time.Time with metav1.Time and fix map types in the generated file
# Add import for metav1 if needed
sed -i 's/time\.Time/metav1.Time/g' apis/common/resources/zz_generated_resources.go
sed -i 's/map\[string\]interface{}/\*map[string]string/g' apis/common/resources/zz_generated_resources.go
sed -i 's|.*"time".*|	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"|' apis/common/resources/zz_generated_resources.go

echo -e "${GREEN}✅ Replacements for time package types and map types completed successfully.${RESET}"

gofmt -w ${OUTPUT_FILE}

echo "Preparing CRD output directory..."
# Clean up and create the output directory
rm -rf "$CRD_OUTPUT_DIR"
mkdir -p "$CRD_OUTPUT_DIR"
echo "CRD output directory is ready at $CRD_OUTPUT_DIR."