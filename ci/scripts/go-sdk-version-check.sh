#!/usr/bin/env bash
# Verify the go-sdk submodule and every go.mod that imports it are pinned to
# the same upstream tag. The submodule is the source of truth for CRD
# generation; the go.mod entries are the source of truth for Go imports.
# These must agree, otherwise the generated CRDs and the compiled types drift.
#
# Usage: go-sdk-version-check.sh <repo-root>

set -euo pipefail

repo_root="${1:?Usage: go-sdk-version-check.sh <repo-root>}"
sub_path="modules/go-sdk"
mod_pkg="github.com/eu-sovereign-cloud/go-sdk"

# Submodule version: must be on an exact tag so the comparison is unambiguous.
sub_ver=$(git -C "${repo_root}/${sub_path}" describe --tags --exact-match HEAD 2>/dev/null) || {
  head=$(git -C "${repo_root}/${sub_path}" rev-parse --short HEAD)
  echo "::error::${sub_path} is not on an exact tag (HEAD=${head})."
  echo "  Pin it to a tagged release, e.g.: make go-sdk-update VERSION=v0.5.0"
  exit 1
}

# Walk every go.mod in the repo (except the submodule's own one) and check
# that any require for ${mod_pkg} matches the submodule's tag.
mismatch=0
while IFS= read -r -d '' gomod; do
  ver=$(awk -v pkg="${mod_pkg}" '
    $1 == pkg && $2 ~ /^v/ { print $2; exit }
  ' "${gomod}")
  [ -n "${ver}" ] || continue
  if [ "${ver}" != "${sub_ver}" ]; then
    rel=${gomod#"${repo_root}/"}
    echo "::error::${rel} pins ${mod_pkg} at ${ver}, but ${sub_path} is at ${sub_ver}"
    mismatch=1
  fi
done < <(find "${repo_root}" -path "${repo_root}/${sub_path}" -prune -o -name go.mod -print0)

if [ "${mismatch}" -ne 0 ]; then
  echo ""
  echo "  Run 'make go-sdk-update VERSION=${sub_ver}' to align go.mod with the submodule,"
  echo "  or 'make go-sdk-update VERSION=<other>' to move both to a different tag."
  exit 1
fi

echo "go-sdk version: ${sub_ver} (submodule and all go.mod files in sync)"
