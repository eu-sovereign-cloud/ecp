#!/usr/bin/env bash
# Host-side initialisation for the .devcontainer compose stack.
#
# Invoked via devcontainer.json "initializeCommand" BEFORE compose up.
# Detects the container backend (podman vs docker) and writes two files:
#
#   .devcontainer/.env                 — dynamic values for compose substitution
#   .devcontainer/compose.override.yml — backend-specific runtime flags
#
# Idempotent: safe to re-run on every devcontainer start / rebuild.

set -euo pipefail

_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_REPO_ROOT="$(cd "${_SCRIPT_DIR}/../.." && pwd)"
_DC_DIR="${_REPO_ROOT}/.devcontainer"

mkdir -p "${_DC_DIR}"

# ---------------------------------------------------------------------------
# Detect backend and resolve dynamic values
# ---------------------------------------------------------------------------

BACKEND="$("${_SCRIPT_DIR}/container-runtime-detect.sh")"
HOST_SOCKET="$("${_SCRIPT_DIR}/container-socket-path.sh" "${BACKEND}")"

# Resolve TOOLS_IMAGE from the Makefile so we stay in sync with .config.mk.
TOOLS_IMAGE="$(make -C "${_REPO_ROOT}" -s print-TOOLS_IMAGE)"

# Ensure the tools image exists locally — mirrors what _tools-ensure-image does
# for -ctzd targets so the devcontainer always starts from a current image.
make -C "${_REPO_ROOT}" _tools-ensure-image

# On Docker, the socket is root:docker (0660).  Pass the socket's GID as
# --group-add so the container user can run docker commands.  On Podman the
# socket is user-owned via userns mapping — no extra group needed.
DOCKER_SOCK_GID=""
if [ "${BACKEND}" = "docker" ] && [ -S "${HOST_SOCKET}" ]; then
  DOCKER_SOCK_GID="$(stat -c '%g' "${HOST_SOCKET}")"
fi

# ---------------------------------------------------------------------------
# Write .env (consumed by compose for ${VAR} substitution)
# ---------------------------------------------------------------------------

cat > "${_DC_DIR}/.env" <<EOF
TOOLS_IMAGE=${TOOLS_IMAGE}
HOST_UID=$(id -u)
HOST_GID=$(id -g)
HOST_WORKSPACE=${_REPO_ROOT}
HOST_SOCKET=${HOST_SOCKET}
DOCKER_SOCK_GID=${DOCKER_SOCK_GID}
EOF

# ---------------------------------------------------------------------------
# Write compose.override.yml (backend-specific runtime flags)
# ---------------------------------------------------------------------------

if [ "${BACKEND}" = "podman" ]; then
  # Podman rootless: userns keep-id so the container UID matches the host.
  # SELinux label=disable allows the container to access the host socket
  # without type-enforcement label errors.
  # The user.slice bind mount makes the host cgroup delegation path visible
  # inside the container so the kind-cgroup-preflight check passes.
  # Note: compose has no cgroupns_mode equivalent; rootless Podman inherits
  # the host cgroup namespace by default, so the bind mount is sufficient.
  cat > "${_DC_DIR}/compose.override.yml" <<'YAML'
services:
  dev:
    userns_mode: "keep-id"
    security_opt:
      - label=disable
    volumes:
      - /sys/fs/cgroup/user.slice:/sys/fs/cgroup/user.slice:ro
YAML
else
  # Docker: run as host uid:gid, add docker socket GID for DinD access.
  cat > "${_DC_DIR}/compose.override.yml" <<'YAML'
services:
  dev:
    user: "${HOST_UID}:${HOST_GID}"
    group_add:
      - "${DOCKER_SOCK_GID}"
YAML
fi

# ---------------------------------------------------------------------------
# Detect stale devcontainer
# ---------------------------------------------------------------------------
# Docker Compose does not recreate containers when an image is rebuilt under
# the same tag. We track the image digest between runs and remove any stale
# container so compose creates a fresh one from the updated image.

_IMAGE_ID=$(docker image inspect --format '{{.Id}}' "${TOOLS_IMAGE}" 2>/dev/null || true)
_ID_FILE="${_DC_DIR}/.image-id"
_PREV_ID=$(cat "${_ID_FILE}" 2>/dev/null || true)

[ -n "${_IMAGE_ID}" ] && printf '%s' "${_IMAGE_ID}" > "${_ID_FILE}"

if [ -n "${_PREV_ID}" ] && [ -n "${_IMAGE_ID}" ] && [ "${_PREV_ID}" != "${_IMAGE_ID}" ]; then
  echo "devcontainer-init: tools image was rebuilt — removing stale container..."
  # Find running containers that were created with TOOLS_IMAGE but now have a
  # stale image digest. Only check running containers to keep the scan fast.
  while IFS= read -r cid; do
    [ -z "${cid}" ] && continue
    _cfg=$(docker inspect --format '{{.Config.Image}}' "${cid}" 2>/dev/null || true)
    if [ "${_cfg}" = "${TOOLS_IMAGE}" ]; then
      docker rm -f "${cid}" 2>/dev/null || true
    fi
  done < <(docker ps -q 2>/dev/null)
fi

echo "devcontainer-init: backend=${BACKEND}  socket=${HOST_SOCKET}  image=${TOOLS_IMAGE}"
