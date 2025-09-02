#!/bin/bash

set -e

excluded_schemas=(
# Error schemas
  "Error"
  "Error400"
  "Error401"
  "Error403"
  "Error404"
  "Error500"
  "ErrorSource"
# Reference schemas
  "Reference"
  "ReferenceURN"
  "ReferenceObject"
)

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <output-file> <spec-file>"
    exit 1
fi

OUTPUT_FILE=$1
SPEC_FILE=$2

echo "Generating Go code from $SPEC_FILE to $OUTPUT_FILE"

${GO_TOOL} -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --generate=types \
  --exclude-schemas=$(IFS=,; echo "${excluded_schemas[*]}") \
  -o ${OUTPUT_FILE} -package v1 ${SPEC_FILE}

# Add import for common package
sed -i --regexp-extended "s/import \(/import \(\n\t\"github\.com\/eu\-sovereign\-cloud\/ecp\/apis\/common\"/g" ${OUTPUT_FILE}

# Post-process the generated code to fix any issues
for schema in "${excluded_schemas[@]}"; do
  if [[ "${schema}" =~ "Error" ]]; then
    # When excluding Error schemas, oapi-codegen leaves behind empty type definitions
    # Remove these empty type definitions
    sed -i --regexp-extended "s/(type|\/\/) ${schema}.*//g" ${OUTPUT_FILE}
  fi

  # Replace references to excluded schemas with types defined in common package
  sed -i "s/${schema}/common.${schema}/g" ${OUTPUT_FILE}
done

gofmt -w ${OUTPUT_FILE}