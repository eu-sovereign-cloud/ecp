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

echo "devcontainer-init: backend=${BACKEND}  socket=${HOST_SOCKET}  image=${TOOLS_IMAGE}"
