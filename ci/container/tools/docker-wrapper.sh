#!/bin/sh
# Thin wrapper around the static Docker CLI binary (docker-bin).
#
# Problem: KIND's docker provider runs `docker info --format '{{json .}}'` and
# checks CPUShares from the JSON. Podman's Docker-compatible API reports
# CPUShares:false on cgroupv2 rootless — even though the cpu controller IS
# delegated — because it checks for the legacy cgroup v1 cpu.shares interface
# rather than cgroupv2's cpu.weight. KIND then refuses to create a cluster.
#
# Fix: for `docker info` calls, patch CPUShares to true in the output.
# This is semantically correct: the cpu controller is available, Podman just
# misreports the capability through the Docker-compat API.
#
# All other docker commands pass through unmodified.

REAL_DOCKER=/usr/local/bin/docker-bin

case "$1" in
  info)
    "$REAL_DOCKER" "$@" | sed 's/"CPUShares":false/"CPUShares":true/'
    ;;
  *)
    exec "$REAL_DOCKER" "$@"
    ;;
esac
