#!/usr/bin/env bash
# Detect the container runtime backend behind the `docker` command.
#
# Detection order:
#   1. CLI check  — podman-docker installs a `docker` that reports as podman.
#   2. Socket path — inside a DinD container the CLI is a static Docker binary,
#      but HOST_SOCKET still contains the real host socket path (e.g.
#      /run/user/1000/podman/podman.sock).
#
# Usage: container-runtime-detect.sh
# Output: prints "podman" or "docker" to stdout

set -euo pipefail

# CLI reports as podman (podman-docker on host)
if docker --version 2>/dev/null | grep -qi podman; then
  echo podman
  exit 0
fi

# Static Docker CLI talking to a Podman daemon (DinD via socket mount)
if [ -n "${HOST_SOCKET:-}" ] && echo "${HOST_SOCKET}" | grep -qi podman; then
  echo podman
  exit 0
fi

echo docker
