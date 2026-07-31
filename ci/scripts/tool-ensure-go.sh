#!/usr/bin/env bash
# Install a Go tool only if the installed binary is missing, wrong version, or
# built for a different GOOS/GOARCH than the current environment.
#
# Usage: tool-ensure-go.sh <binary-name> <go-package> <version>
#
# Uses `go version -m <binary>` to read the module version and platform
# embedded at build time. This is universal — no per-tool --version output
# parsing needed. Reinstalls when the binary was built for another OS/arch
# (e.g. host-installed Darwin tools bind-mounted into a Linux *-ctzd container).
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

want_goos="$(go env GOOS)"
want_goarch="$(go env GOARCH)"

if [ -x "${binary}" ]; then
  meta=$(go version -m "${binary}" 2>/dev/null || true)
  installed=$(printf '%s\n' "${meta}" | awk '/^\tmod\t/{print $3; exit}')
  installed="${installed:-unknown}"
  got_goos=$(printf '%s\n' "${meta}" | awk -F= '/^\tbuild\tGOOS=/{print $2; exit}')
  got_goarch=$(printf '%s\n' "${meta}" | awk -F= '/^\tbuild\tGOARCH=/{print $2; exit}')

  platform_ok=1
  if [ -z "${got_goos}" ] || [ -z "${got_goarch}" ] \
    || [ "${got_goos}" != "${want_goos}" ] || [ "${got_goarch}" != "${want_goarch}" ]; then
    platform_ok=0
  fi

  if [ "${installed}" = "${version}" ] && [ "${platform_ok}" -eq 1 ]; then
    echo "  ${name} ${version} (up to date)"
    exit 0
  fi

  if [ "${platform_ok}" -eq 0 ]; then
    got_platform="${got_goos:-unknown}/${got_goarch:-unknown}"
    echo "  ${name}: ${installed} (${got_platform}) -> ${version} (${want_goos}/${want_goarch})"
  else
    echo "  ${name}: ${installed} -> ${version}"
  fi
else
  echo "  ${name}: not installed"
fi

echo "  Installing ${name} ${version}..."
GOBIN="${bin_dir}" go install "${pkg}@${version}"
echo "  ${name} ${version} installed"
