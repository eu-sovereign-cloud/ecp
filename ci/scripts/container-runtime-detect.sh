#!/usr/bin/env bash
# Detect the backend behind the `docker` command.
# podman-docker installs a `docker` script/alias that uses podman as backend.
#
# Usage: container-runtime-detect.sh
# Output: prints "podman" or "docker" to stdout

set -euo pipefail

if docker --version 2>/dev/null | grep -qi podman; then
  echo podman
else
  echo docker
fi
