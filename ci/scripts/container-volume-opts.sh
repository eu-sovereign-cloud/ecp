#!/usr/bin/env bash
# Output the SELinux volume option for podman.
#
# Usage: container-volume-opts.sh <backend>
#   backend: "podman" or "docker" (output of container-runtime-detect.sh)
#
# Output: "z" for podman, empty string for docker
# Note: lowercase "z" = shared label (multiple containers can access the volume).
#       uppercase "Z" = private label (only one container) — breaks when the
#       same bind mount is used by both persistent and ephemeral containers.
# No leading colon — callers compose the full option string.

set -euo pipefail

backend="${1:?Usage: container-volume-opts.sh <backend>}"

if [[ "${backend}" == "podman" ]]; then
  echo "z"
fi
