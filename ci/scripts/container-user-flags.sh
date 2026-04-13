#!/usr/bin/env bash
# Output the docker run flags for user mapping, depending on whether the
# docker backend is real Docker or Podman.
#
# Usage: container-user-flags.sh <backend>
#   backend: "podman" or "docker" (output of container-runtime-detect.sh)
#
# Output: space-separated flags to stdout
#   podman CLI  (podman-docker): --userns=keep-id
#   docker CLI → podman daemon:  --userns=host
#   docker CLI → docker daemon:  --user=<uid>:<gid>

set -euo pipefail

backend="${1:?Usage: container-user-flags.sh <backend>}"

if [[ "${backend}" == "podman" ]]; then
  if docker --version 2>/dev/null | grep -qi podman; then
    # Native Podman CLI (podman-docker on host)
    echo "--userns=keep-id"
  else
    # Static Docker CLI talking to Podman daemon (DinD via socket mount)
    echo "--userns=host"
  fi
else
  echo "--user=$(id -u):$(id -g)"
fi
