#!/usr/bin/env bash
# Report whether a container image's build inputs differ from the tree state
# that produced the pinned builder digest.
#
# CI rebuilds the builder and bumps .builder-digest whenever builder inputs
# change, so the commit that last modified .builder-digest marks the last
# "blessed" state of the builder -> tools -> dev image stack. If the working
# tree's inputs for <image> differ from that commit, the pinned remote builder
# (or a cached local tools/dev image) is stale and must be rebuilt locally.
#
# Usage: image-inputs-changed.sh <builder|tools|dev>
# Output: prints "changed" when inputs differ, nothing otherwise.
# Exit: 0 (result on stdout); 2 on usage error.
#
# Fail-safe: not a git repo -> prints nothing ("unchanged"); a git repo whose
# digest commit is unreachable (shallow clone) -> prints "changed".
set -uo pipefail

image="${1:?Usage: image-inputs-changed.sh <builder|tools|dev>}"

repo_root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
cd "${repo_root}" || exit 0

# Image-CONTENT paths only. Each layer includes every lower layer. .github/**
# is intentionally excluded — it changes CI behaviour, not what `docker build`
# produces locally (CI's broader set lives in paths-filter-gen.sh, by design).
builder_paths=(ci/container/builder .config.mk .common.mk Makefile ci/tools ci/scripts)
tools_paths=("${builder_paths[@]}" ci/container/tools)
dev_paths=("${tools_paths[@]}" ci/container/dev)

case "${image}" in
  builder) paths=("${builder_paths[@]}") ;;
  tools)   paths=("${tools_paths[@]}") ;;
  dev)     paths=("${dev_paths[@]}") ;;
  *) echo "error: unknown image '${image}' (expected builder, tools or dev)" >&2; exit 2 ;;
esac

digest_commit=$(git log -1 --format=%H -- .builder-digest 2>/dev/null)
if [ -z "${digest_commit}" ] || \
   ! git rev-parse --verify --quiet "${digest_commit}^{commit}" >/dev/null 2>&1; then
  echo changed; exit 0
fi

if ! git diff --quiet "${digest_commit}" -- "${paths[@]}" 2>/dev/null; then
  echo changed; exit 0
fi
if [ -n "$(git ls-files --others --exclude-standard -- "${paths[@]}" 2>/dev/null)" ]; then
  echo changed
fi
exit 0
