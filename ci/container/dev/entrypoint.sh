#!/usr/bin/env bash
# Entrypoint for the persistent dev container.
# Dual-mode:
#   - Root (Docker): creates dev user matching host UID:GID, starts sshd as root
#   - Non-root (Podman keep-id): starts sshd as current user
set -euo pipefail

: "${HOST_UID:?HOST_UID must be set}"
: "${HOST_GID:?HOST_GID must be set}"

DEV_USER="dev"
SSHD_DIR="/workspace/.cache/sshd"
DEV_HOME="/workspace/.cache/container-home"

setup_ssh_keys() {
  local auth_keys_dir="$1"
  mkdir -p "${auth_keys_dir}"

  # Generate host keys if not present (persistent across container restarts)
  if [ ! -f "${SSHD_DIR}/ssh_host_ed25519_key" ]; then
    ssh-keygen -t ed25519 -f "${SSHD_DIR}/ssh_host_ed25519_key" -N ""
    ssh-keygen -t rsa -b 4096 -f "${SSHD_DIR}/ssh_host_rsa_key" -N ""
  fi

  # Collect authorized keys from mounted host ~/.ssh
  local auth_keys="${auth_keys_dir}/authorized_keys"
  : > "${auth_keys}"

  if [ -f /tmp/host-ssh/authorized_keys ]; then
    cat /tmp/host-ssh/authorized_keys >> "${auth_keys}"
  else
    # No authorized_keys file — gather all public keys
    for pub in /tmp/host-ssh/*.pub; do
      [ -f "${pub}" ] && cat "${pub}" >> "${auth_keys}"
    done
  fi

  chmod 600 "${auth_keys}"
}

write_sshd_config() {
  local auth_keys_file="$1"
  local home_dir="$2"
  cat > "${SSHD_DIR}/sshd_config" <<EOF
Port 2222
ListenAddress 0.0.0.0
HostKey ${SSHD_DIR}/ssh_host_ed25519_key
HostKey ${SSHD_DIR}/ssh_host_rsa_key
AuthorizedKeysFile ${auth_keys_file}
StrictModes no
PasswordAuthentication no
PubkeyAuthentication yes
PubkeyAcceptedAlgorithms +ssh-rsa
UsePAM no
# Force HOME to our well-known cache dir regardless of what passwd says.
# This keeps all shell state out of the repo root and in the gitignored .cache/.
SetEnv HOME=${home_dir}
Subsystem sftp /usr/lib/openssh/sftp-server
PidFile ${SSHD_DIR}/sshd.pid
EOF
}

setup_env() {
  local profile_dir="$1"
  mkdir -p "${profile_dir}"
  # HOST_WORKSPACE is expanded now (from the container's environment) so that
  # SSH login shells can use the real host path for nested docker volume mounts.
  cat > "${profile_dir}/dev-env.sh" <<EOF
export GOPATH=/workspace/.cache/go
export GOCACHE=/workspace/.cache/go-build
export PATH="/workspace/ci/tools/bin:/usr/local/go/bin:/usr/local/bin:\${PATH}"
export HOST_WORKSPACE="${HOST_WORKSPACE:-/workspace}"
export HOST_SOCKET="${HOST_SOCKET:-/var/run/docker.sock}"
cd /workspace 2>/dev/null || true
EOF
}

mkdir -p "${SSHD_DIR}" "${DEV_HOME}"

if [ "$(id -u)" -eq 0 ]; then
  # ── Docker mode: running as real root ──────────────────────────────
  if ! getent group "${HOST_GID}" >/dev/null 2>&1; then
    groupadd -g "${HOST_GID}" "${DEV_USER}"
  fi

  if ! id "${DEV_USER}" >/dev/null 2>&1; then
    useradd -m -u "${HOST_UID}" -g "${HOST_GID}" -s /bin/bash "${DEV_USER}"
  else
    usermod -u "${HOST_UID}" -g "${HOST_GID}" "${DEV_USER}" 2>/dev/null || true
  fi

  # Unlock the account so sshd allows pubkey login.
  # useradd sets password to '!' (locked); OpenSSH 10+ rejects locked accounts.
  passwd -d "${DEV_USER}" >/dev/null 2>&1

  # Grant access to the Docker socket for DinD via socket mount.
  # The socket's GID on the host may differ from any group inside the container,
  # so we detect it at runtime and add the dev user to the matching group.
  if [ -S /var/run/docker.sock ]; then
    SOCK_GID=$(stat -c '%g' /var/run/docker.sock)
    if ! getent group "${SOCK_GID}" >/dev/null 2>&1; then
      groupadd -g "${SOCK_GID}" docker-host
    fi
    usermod -aG "$(getent group "${SOCK_GID}" | cut -d: -f1)" "${DEV_USER}"
  fi

  echo "${DEV_USER} ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/"${DEV_USER}"

  setup_ssh_keys "${DEV_HOME}/.ssh"
  write_sshd_config "${DEV_HOME}/.ssh/authorized_keys" "${DEV_HOME}"
  setup_env "/etc/profile.d" "${DEV_HOME}"

  chmod 700 "${DEV_HOME}/.ssh"
  chown -R "${HOST_UID}:${HOST_GID}" "${DEV_HOME}" "${SSHD_DIR}"

  exec /usr/sbin/sshd -D -e -f "${SSHD_DIR}/sshd_config"
else
  # ── Podman keep-id mode: running as host user ─────────────────────
  # HOME is forced to DEV_HOME via sshd's SetEnv, so all shell state
  # lands in /workspace/.cache/container-home (gitignored) regardless
  # of what Podman puts in /etc/passwd.
  setup_ssh_keys "${DEV_HOME}/.ssh"
  write_sshd_config "${DEV_HOME}/.ssh/authorized_keys" "${DEV_HOME}"
  setup_env "${DEV_HOME}/.profile.d" "${DEV_HOME}"

  # Write .profile into DEV_HOME. sshd's SetEnv HOME=DEV_HOME ensures
  # the login shell finds it there, and upgrades /bin/sh to bash.
  cat > "${DEV_HOME}/.profile" <<'PROFILE'
if [ -z "${BASH_VERSION:-}" ]; then
  exec /bin/bash -l
fi

# Source profile.d scripts
for f in /etc/profile.d/*.sh "${HOME}/.profile.d"/*.sh; do
  [ -r "${f}" ] && . "${f}"
done
PROFILE

  chmod 700 "${DEV_HOME}/.ssh"

  exec /usr/sbin/sshd -D -e -f "${SSHD_DIR}/sshd_config"
fi
