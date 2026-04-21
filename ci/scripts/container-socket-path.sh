#!/usr/bin/env bash
# Output the host container runtime socket path.
#
# Usage: container-socket-path.sh <backend>
#   backend: "podman" or "docker" (output of container-runtime-detect.sh)
#
# Output: absolute path to the socket file
# Exit code: 1 if socket not found

set -euo pipefail

backend="${1:?Usage: container-socket-path.sh <backend>}"

if [[ "${backend}" == "podman" ]]; then
  socket="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/podman/podman.sock"
else
  socket="/var/run/docker.sock"
fi

if [[ ! -S "${socket}" ]]; then
  echo "container-socket-path.sh: socket not found: ${socket}" >&2
  exit 1
fi

echo "${socket}"
