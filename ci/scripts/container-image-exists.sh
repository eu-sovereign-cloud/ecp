#!/usr/bin/env bash
# Check if a container image exists locally.
#
# Usage: container-image-exists.sh <image>
# Output: prints "yes" if the image exists, nothing otherwise
# Exit code: always 0

set -euo pipefail

image="${1:?Usage: container-image-exists.sh <image>}"

if docker image inspect "${image}" >/dev/null 2>&1; then
  echo yes
fi
