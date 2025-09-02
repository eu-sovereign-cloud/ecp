#!/bin/bash

BLUE="\033[1;34m"
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
RESET="\033[0m"

template_file="scripts/common-template/dummy.yaml.tpl"
out_dir="spec/spec"
schemas_dir="./schemas"

set -e

gomplate --version >/dev/null || {
  echo -e "${YELLOW}⚠️  gomplate command not found or not executable. Please install gomplate.\n    For instructions visit ${BLUE}https://docs.gomplate.ca/installing/${RESET}";
  exit 1;
}

echo "Creating output directory at ${out_dir} if it doesn't exist..."
mkdir -p ${out_dir}

echo "Generating OpenAPI dummy spec files for common models..."
echo '{"dir": "'${schemas_dir}'", "file": "errors.yaml"}' | gomplate -f ${template_file} -o ${out_dir}/dummy-errors-spec.yaml -d data=spec/spec/schemas/errors.yaml -d path=stdin:path.json
echo '{"dir": "'${schemas_dir}'", "file": "resource.yaml"}' | gomplate -f ${template_file} -o ${out_dir}/dummy-resources-spec.yaml -d data=spec/spec/schemas/resource.yaml -d path=stdin:path.json
#echo '{"dir": "'${schemas_dir}'", "file": "parameters.yaml"}' | gomplate -f ${template_file} -o ${out_dir}/dummy-parameters-spec.yaml -d data=spec/spec/schemas/parameters.yaml -d path=stdin:path.json

echo -e "${GREEN}✅ Dummy OpenAPI spec files generated successfully.${RESET}"
echo "Bundling dummy OpenAPI spec files into a single file with models..."

schemas=$(find ${out_dir}/schemas -type f)

npx @redocly/cli bundle --remove-unused-components ${out_dir}/dummy-errors-spec.yaml -o ${out_dir}/errors-bundled.yaml
npx @redocly/cli bundle --remove-unused-components ${out_dir}/dummy-resources-spec.yaml -o ${out_dir}/resources-bundled.yaml
#npx @redocly/cli bundle ${out_dir}/dummy-parameters-spec.yaml -o ${out_dir}/parameters-bundled.yaml

echo -e "${GREEN}✅ Bundling completed successfully.${RESET}"

echo "Cleaning up dummy spec files..."
rm ${out_dir}/dummy-*-spec.yaml

excluded_schemas=$(cat ${out_dir}/errors-bundled.yaml ${out_dir}/resources-bundled.yaml | grep "Excluded:" | sed 's/://g' | tr '\n' ',' | tr -s ' ' | sed 's/ //g')
excluded_schemas=${excluded_schemas%,}

echo "Generating Go code from bundled OpenAPI spec files for common models..."

mkdir -p apis/common/errors
mkdir -p apis/common/resources
#mkdir -p apis/common/parameters

${GO_TOOL} -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --generate=types \
  --exclude-schemas=${excluded_schemas} \
  -o apis/common/errors/zz_generated_errors.go -package errors ${out_dir}/errors-bundled.yaml

${GO_TOOL} -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --generate=types \
  --exclude-schemas=${excluded_schemas} \
  -o apis/common/resources/zz_generated_resources.go -package resources ${out_dir}/resources-bundled.yaml

#${GO_TOOL} -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --generate=types \
#  --exclude-schemas=${excluded_schemas} \
#  -o apis/common/parameters/zz_generated_parameters.go -package parameters ${out_dir}/parameters-bundled.yaml

echo -e "${GREEN}✅ Go code generated successfully.${RESET}"