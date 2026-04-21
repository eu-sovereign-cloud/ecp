#!/usr/bin/env bash
# Verify that the cgroupv2 'cpuset' controller is delegated to the current
# user's systemd session. KIND requires cpuset for kubelet CPU topology
# management; without it, 'kind create cluster' fails with:
#   "running kind with rootless provider requires setting systemd property Delegate=yes"
#
# This check only applies to Podman rootless on cgroupv2. Docker manages cgroups
# through its daemon and KIND uses a different code path that does not inspect
# user-session delegation — on Docker hosts the check is skipped entirely.
#
# Usage: kind-cgroup-preflight.sh
#   Exits 0 if the check passes, is skipped (non-cgroupv2 or non-Podman), or
#   prints a remediation message and exits 1 if cpuset is missing.

set -euo pipefail

# Not a cgroupv2 (unified) system — nothing to check.
if [ ! -f /sys/fs/cgroup/cgroup.controllers ]; then
  exit 0
fi

# Determine the container backend. Prefer the value propagated from the host
# Makefile via the HOST_SOCKET env var (already present in every container),
# then fall back to running the detection script directly.
_detect_backend() {
  # Static Docker CLI talking to a Podman daemon (DinD via socket mount)
  if [ -n "${HOST_SOCKET:-}" ] && echo "${HOST_SOCKET}" | grep -qi podman; then
    echo podman
    return
  fi
  # docker --version reports as podman on podman-docker installations
  if docker --version 2>/dev/null | grep -qi podman; then
    echo podman
    return
  fi
  echo docker
}

_backend=$(_detect_backend)

# cpuset delegation is only a concern for Podman rootless.
if [ "${_backend}" != "podman" ]; then
  exit 0
fi

_controllers="/sys/fs/cgroup/user.slice/user-$(id -u).slice/user@$(id -u).service/cgroup.controllers"

if grep -qw cpuset "${_controllers}" 2>/dev/null; then
  exit 0
fi

echo "ERROR: cgroup 'cpuset' controller not delegated to user $(id -u) — KIND will fail." >&2
echo "" >&2
echo "Run on the host (once) and then re-login:" >&2
echo "  sudo mkdir -p /etc/systemd/system/user@.service.d/" >&2
echo "  printf '[Service]\\nDelegate=yes\\n' | sudo tee /etc/systemd/system/user@.service.d/delegate.conf" >&2
echo "  sudo systemctl daemon-reload" >&2
echo "" >&2
echo "Verify with: cat ${_controllers}" >&2
exit 1
