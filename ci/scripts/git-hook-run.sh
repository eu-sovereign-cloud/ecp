#!/usr/bin/env bash
# Dispatch a git hook to its Makefile validation gate.
#
# Run by the pre-commit and pre-push hooks in .githooks/. Detects whether git
# is running on the host or inside an ECP build container and picks the form:
#   host           -> make <target>-ctzd   (gate runs in the tools container)
#   ECP container  -> make <target>        (gate runs directly; ECP_CONTAINER set)
#
# Usage: git-hook-run.sh <make-target> <skip-config-key>
#   <make-target>      Makefile gate to run (e.g. "pre-commit", "pre-merge")
#   <skip-config-key>  git config key that, when "true", skips the gate
#                      (e.g. "hooks.skipPreCommit")
#
# Exit code: the wrapped `make` exit code, or 0 when skipped.

set -euo pipefail

target="${1:?Usage: git-hook-run.sh <make-target> <skip-config-key>}"
skip_key="${2:?Usage: git-hook-run.sh <make-target> <skip-config-key>}"

cd "$(git rev-parse --show-toplevel)"

if [ "$(git config --bool --get "${skip_key}" 2>/dev/null || true)" = "true" ]; then
  echo "==> ${skip_key}=true — skipping 'make ${target}'."
  exit 0
fi

# ECP_CONTAINER is baked into the builder image (inherited by tools and dev).
# Absent => running on the host => dispatch into the tools container via -ctzd.
if [ -n "${ECP_CONTAINER:-}" ]; then
  suffix=""
else
  suffix="-ctzd"
fi

echo "==> git hook: running 'make ${target}${suffix}'"
exec make "${target}${suffix}"
