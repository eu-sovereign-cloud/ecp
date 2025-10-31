#!/bin/bash
# generate-all-models.sh
set -euo pipefail

GREEN="\033[1;32m"
RESET="\033[0m"

SCHEMA_DIR="foundation/delegator/go-sdk/pkg/spec/schema"
OUTPUT_ROOT="foundation/delegator/api/types"

if [ ! -d "${SCHEMA_DIR}" ]; then
  echo "Schema directory ${SCHEMA_DIR} not found" >&2
  exit 1
fi

GENERATED_DIRS=()

process_file () {
  local src="$1"
  local base
  base=$(basename "${src}")

  local out_dir="${OUTPUT_ROOT}"
  local out_file="${out_dir}/zz_generated_${base}"

  mkdir -p "${out_dir}"
  cp "${src}" "${out_file}"

  # Remove existing package line
  sed -i '/^package[[:space:]].*/d' "${out_file}"

  # Prepend new package + kubebuilder annotations
  {
    echo "package types"
    echo ""
    echo "// +kubebuilder:object:generate=true"
    echo "// +kubebuilder:object:root=true"
    echo ""
  } | cat - "${out_file}" > "${out_file}.tmp" && mv "${out_file}.tmp" "${out_file}"

  # time.Time -> metav1.Time
  if grep -q 'time.Time' "${out_file}"; then
    sed -i 's/\btime\.Time\b/metav1.Time/g' "${out_file}"
    if grep -q '\"time\"' "${out_file}"; then
      sed -i 's|.*"time".*|	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"|' "${out_file}"
    elif ! grep -q 'k8s.io/apimachinery/pkg/apis/meta/v1' "${out_file}"; then
      if ! grep -q "import (" "${out_file}"; then
        sed -i "/^package types/a import (\n\tmetav1 \"k8s.io/apimachinery/pkg/apis/meta/v1\"\n)" "${out_file}"
      else
        sed -i "/import (/a \\\tmetav1 \"k8s.io/apimachinery/pkg/apis/meta/v1\"" "${out_file}"
      fi
    fi
  fi

  # Map type fix
  sed -i 's/map\[string\]interface{}/\*map[string]string/g' "${out_file}"

  gofmt -w "${out_file}"

  GENERATED_DIRS+=("${out_dir}")
  echo -e "${GREEN}✅ Generated ${out_file}${RESET}"
}

echo "Scanning ${SCHEMA_DIR}..."
for f in "${SCHEMA_DIR}"/*.go; do
  [ -e "$f" ] || continue
  process_file "$f"
done

echo "Running controller-gen for DeepCopy..."
go run sigs.k8s.io/controller-tools/cmd/controller-gen object paths="./${OUTPUT_ROOT}"
echo -e "${GREEN}✅ All models processed.${RESET}"
