#!/usr/bin/env bash
# Output the docker run flags for user mapping and SELinux volume labeling,
# depending on whether the docker backend is real Docker or podman.
#
# Usage: container-user-flags.sh <backend>
#   backend: "podman" or "docker" (output of container-runtime-detect.sh)
#
# Output: space-separated flags to stdout
#   For podman: --userns=keep-id
#   For docker: --user=<uid>:<gid>

set -euo pipefail

backend="${1:?Usage: container-user-flags.sh <backend>}"

if [[ "${backend}" == "podman" ]]; then
  echo "--userns=keep-id"
else
  echo "--user=$(id -u):$(id -g)"
fi
