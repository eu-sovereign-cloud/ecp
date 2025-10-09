#!/bin/bash

GREEN="\033[1;32m"
RESET="\033[0m"

set -e

if [ "$1" = "help" ]; then
    echo "Usage: $0 <api-name> <api-version> <common-model1> [<common-model2> ...]"
    echo "Example: $0 identity v1alpha1 resource errors"
    echo "Available common models: 'errors', 'resource'"
    echo "Source file is expected to be at 'internal/go-sdk/<api-name>.go'"
    echo "Output directory will be './apis/generated/types/<api-name>/<api-version>' (it will be created if it doesn't exist)."
    exit 0
fi

if [ "$#" -lt 3 ]; then
    echo "Usage: $0 <api-name> <api-version> <common-model1> [<common-model2> ...]"
    echo "Available common models: 'errors', 'resource'"
    exit 1
fi

API_NAME=$1
API_VERSION=$2
COMMON_MODELS=( ${@:3} )
SOURCE_FILE="internal/go-sdk/pkg/spec/schema/${API_NAME}.go"
OUTPUT_DIR="./apis/generated/types/${API_NAME}/${API_VERSION}"
OUTPUT_FILE="${OUTPUT_DIR}/zz_generated_${API_NAME}.go"

echo "Copying Go code from ${SOURCE_FILE} to ${OUTPUT_FILE}"

mkdir -p "${OUTPUT_DIR}"
cp "${SOURCE_FILE}" "${OUTPUT_FILE}"

# Remove the original package declaration
sed -i '/^package .*/d' "${OUTPUT_FILE}"

# Add the new package declaration at the top of the file
echo "package ${API_VERSION}" | cat - "${OUTPUT_FILE}" > temp && mv temp "${OUTPUT_FILE}"


for model in "${COMMON_MODELS[@]}"; do
    schemas=$(grep --perl-regexp '^(?!\/\/)type' "./apis/generated/types/${model}/zz_generated_${model}.go" | tr -s ' ' | cut -d ' ' -f "2" | tr '\n' ',')
    schemas=${schemas%,}
    schemas_array=( $(echo "${schemas}" | tr ',' ' ') )


    requires_import=false

    echo "Post-processing the Go code to replace excluded schemas with references to ${model} package..."
    for schema in "${schemas_array[@]}"; do
          echo "schema are: ${schema}"
        # Remove potential duplicate type definitions
        sed -i --regexp-extended "/^type ${schema} struct \{/,/^\}$/d" "${OUTPUT_FILE}"
        if [ $(grep -c "${schema}" ${OUTPUT_FILE}) -gt 0 ]; then
            requires_import=true
        fi

        sed -i "/^\/\//! s/ ${schema}/ ${model}.${schema}/g" ${OUTPUT_FILE}
        sed -i "/^\/\//! s/\*${schema}/*${model}.${schema}/g" ${OUTPUT_FILE}
        sed -i "/^\/\//! s/\]${schema}/]${model}.${schema}/g" ${OUTPUT_FILE}
    done

    if [ "$requires_import" = true ]; then
        # Check if an import block already exists
        if ! grep -q "import (" "${OUTPUT_FILE}"; then
            # Add a new import block if it doesn't exist
            sed -i "/^package ${API_VERSION}/a \\import (\\n\\t${model} \"github.com/eu-sovereign-cloud/ecp/apis/generated/types/${model}\"\\n)" "${OUTPUT_FILE}"
        else
            # Add to the existing import block
            sed -i "/import (/{a\\
\\t${model} \"github.com/eu-sovereign-cloud/ecp/apis/generated/types/${model}\"
}" "${OUTPUT_FILE}"
        fi
    fi

    echo -e "${GREEN}✅ Post-processing for types/${model} completed successfully.${RESET}\n"
done

echo "Replacing time package types with k8s metav1 package time types and fixing map types in the generated file..."

# Replace time.Time with metav1.Time and fix map types in the generated file
# Add import for metav1 if needed
sed -i 's/time\.Time/metav1.Time/g' ${OUTPUT_FILE}
sed -i 's/map\[string\]interface{}/\*map[string]string/g' ${OUTPUT_FILE}

# Replace time import with metav1 import, or add it if no time import exists but time.Time was replaced.
if grep -q "metav1.Time" "${OUTPUT_FILE}"; then
    if grep -q '.*"time".*' "${OUTPUT_FILE}"; then
        sed -i 's|.*"time".*|	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"|' "${OUTPUT_FILE}"
    elif ! grep -q '.*"k8s.io/apimachinery/pkg/apis/meta/v1".*' "${OUTPUT_FILE}"; then
        if ! grep -q "import (" "${OUTPUT_FILE}"; then
             sed -i "/^package ${API_VERSION}/a \\nimport (\\n\\tmetav1 \"k8s.io/apimachinery/pkg/apis/meta/v1\"\\n)" "${OUTPUT_FILE}"
        else
             sed -i "/import (/{a\\
\\tmetav1 \"k8s.io/apimachinery/pkg/apis/meta/v1\"
}" "${OUTPUT_FILE}"
        fi
    fi
fi


echo -e "${GREEN}✅ Replacements for time package types and map types completed successfully.${RESET}\n"

gofmt -w ${OUTPUT_FILE}

echo "Add +kubebuilder:object:root=true annotation to ${OUTPUT_FILE}..."
sed -i "/^package ${API_VERSION}/a\// +kubebuilder:object:generate=true" "${OUTPUT_FILE}"
echo -e "${GREEN}✅ Annotations added successfully.${RESET}\n"

echo "Generating DeepCopy methods for ${API_NAME} models..."
${GO_TOOL} -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object paths="${OUTPUT_DIR}"

echo -e "${GREEN}✅ DeepCopy methods generated successfully.${RESET}"
echo -e "${GREEN}✅ All tasks completed successfully for ${API_NAME} ${API_VERSION} models!${RESET}\n"
