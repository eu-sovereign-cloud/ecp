#!/usr/bin/env bash
# Verify that the cgroupv2 'cpuset' controller is delegated to the current
# user's systemd session. KIND requires cpuset for kubelet CPU topology
# management; without it, 'kind create cluster' fails with:
#   "running kind with rootless provider requires setting systemd property Delegate=yes"
#
# Usage: kind-cgroup-preflight.sh
#   Exits 0 if the check passes (or if not on a cgroupv2 system).
#   Exits 1 with a clear error and remediation steps if cpuset is missing.

set -euo pipefail

# Not a cgroupv2 (unified) system — nothing to check.
if [ ! -f /sys/fs/cgroup/cgroup.controllers ]; then
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
