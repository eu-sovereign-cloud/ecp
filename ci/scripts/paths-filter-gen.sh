#!/usr/bin/env bash
# Emit the dorny/paths-filter configuration consumed by CI: a `builder` filter
# (builder-image input paths) plus one filter per CI-relevant Go module.
#
# Usage: paths-filter-gen.sh
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Builder-image input paths (dorny/paths-filter globs) — single source of truth
# for CI builder-change detection. Includes .github/** build-logic paths: a
# change to HOW CI builds the image must still re-trigger a rebuild + full
# validation. (image-inputs-changed.sh uses an image-content-only subset for
# LOCAL detection, by design.)
builder_paths=(
  'ci/container/builder/**'
  'ci/tools/**'
  'ci/scripts/**'
  '.config.mk'
  '.common.mk'
  'Makefile'
  '.github/workflows/builder-publish.yaml'
  '.github/workflows/pre-merge.yaml'
  '.github/actions/builder-build-push/**'
  '.github/actions/builder-pr-publish/**'
)

printf 'builder:\n'
for p in "${builder_paths[@]}"; do printf '  - %s\n' "${p}"; done

while IFS= read -r m; do
  [ -n "${m}" ] || continue
  printf '%s:\n  - %s/**\n' "${m}" "${m}"
done < <("${script_dir}/go-modules.sh")
