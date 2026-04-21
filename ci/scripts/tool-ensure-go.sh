#!/usr/bin/env bash
# Install a Go tool only if the installed version doesn't match the desired one.
#
# Usage: tool-ensure-go.sh <binary-name> <go-package> <version>
#
# Uses `go version -m <binary>` to read the module version embedded at build
# time. This is universal — no per-tool --version output parsing needed.
#
# GOBIN must be set by the caller so both the check and the install resolve the
# binary in the same directory.
set -euo pipefail

name="${1:?Usage: tool-ensure-go.sh <name> <package> <version>}"
pkg="${2:?}"
version="${3:?}"

bin_dir="${GOBIN:-$(go env GOBIN)}"
bin_dir="${bin_dir:-$(go env GOPATH)/bin}"
binary="${bin_dir}/${name}"

if [ -x "${binary}" ]; then
  installed=$(go version -m "${binary}" 2>/dev/null \
    | awk '/^\tmod\t/{print $3}' || echo "unknown")
  if [ "${installed}" = "${version}" ]; then
    echo "  ${name} ${version} (up to date)"
    exit 0
  fi
  echo "  ${name}: ${installed} -> ${version}"
else
  echo "  ${name}: not installed"
fi

echo "  Installing ${name} ${version}..."
GOBIN="${bin_dir}" go install "${pkg}@${version}"
echo "  ${name} ${version} installed"
