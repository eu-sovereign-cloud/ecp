#!/usr/bin/env bash
# Output security options needed for the container runtime.
#
# Usage: container-security-opts.sh <backend>
#   backend: "podman" or "docker" (output of container-runtime-detect.sh)
#
# For podman on SELinux-enabled hosts, disables SELinux confinement so the
# container can access the host socket (docker-in-docker) without label errors.

set -euo pipefail

backend="${1:?Usage: container-security-opts.sh <backend>}"

if [[ "${backend}" == "podman" ]]; then
  echo "--security-opt label=disable"
fi
