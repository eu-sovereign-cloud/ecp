#!/usr/bin/env bash
# Ensure a k6 runner is available and write scripts/.k6-cmd for run-k6.sh.
#
# Preference order:
#   1. Native k6 on PATH (any version >= K6_MIN_VERSION)
#   2. Docker image K6_IMAGE (default grafana/k6:0.57.0)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOAD_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
CMD_FILE="${SCRIPT_DIR}/.k6-cmd"

# Minimum native version. Docker fallback is always the pinned image.
K6_MIN_VERSION="${K6_MIN_VERSION:-0.54.0}"
K6_IMAGE="${K6_IMAGE:-grafana/k6:0.57.0}"

# version_ge A B — true if A >= B (semver major.minor.patch, ignores pre-release).
version_ge() {
	local a="$1" b="$2"
	local a_maj a_min a_pat b_maj b_min b_pat
	IFS=. read -r a_maj a_min a_pat <<<"${a%%-*}"
	IFS=. read -r b_maj b_min b_pat <<<"${b%%-*}"
	a_maj=${a_maj:-0} a_min=${a_min:-0} a_pat=${a_pat:-0}
	b_maj=${b_maj:-0} b_min=${b_min:-0} b_pat=${b_pat:-0}
	if ((a_maj != b_maj)); then ((a_maj > b_maj)); return; fi
	if ((a_min != b_min)); then ((a_min > b_min)); return; fi
	((a_pat >= b_pat))
}

parse_k6_version() {
	# "k6 v0.57.0 (...)" or "k6 v2.1.0 (...)" → 0.57.0 / 2.1.0
	sed -n 's/^k6 v\([0-9][0-9.]*\).*/\1/p' | head -1
}

write_cmd() {
	# Single line: either "native|/path/to/k6" or "docker|image"
	printf '%s\n' "$1" >"${CMD_FILE}"
}

try_native() {
	if ! command -v k6 >/dev/null 2>&1; then
		return 1
	fi
	local bin ver
	bin="$(command -v k6)"
	ver="$("${bin}" version 2>/dev/null | parse_k6_version || true)"
	if [[ -z "${ver}" ]]; then
		echo "k6-start: found ${bin} but could not parse version" >&2
		return 1
	fi
	if ! version_ge "${ver}" "${K6_MIN_VERSION}"; then
		echo "k6-start: native k6 ${ver} is older than minimum ${K6_MIN_VERSION}" >&2
		return 1
	fi
	write_cmd "native|${bin}"
	echo "k6-start: using native k6 ${ver} (${bin})"
	return 0
}

try_docker() {
	if ! command -v docker >/dev/null 2>&1; then
		echo "k6-start: docker not found" >&2
		return 1
	fi
	if ! docker info >/dev/null 2>&1; then
		echo "k6-start: docker is not running" >&2
		return 1
	fi
	echo "k6-start: pulling ${K6_IMAGE} (if needed)..."
	if ! docker pull "${K6_IMAGE}" >/dev/null; then
		echo "k6-start: failed to pull ${K6_IMAGE}" >&2
		return 1
	fi
	# Smoke-check the image.
	if ! docker run --rm "${K6_IMAGE}" version >/dev/null; then
		echo "k6-start: ${K6_IMAGE} failed to run" >&2
		return 1
	fi
	write_cmd "docker|${K6_IMAGE}"
	local ver
	ver="$(docker run --rm "${K6_IMAGE}" version 2>/dev/null | parse_k6_version || echo unknown)"
	echo "k6-start: using docker k6 ${ver} (${K6_IMAGE})"
	return 0
}

if try_native; then
	exit 0
fi

echo "k6-start: native k6 unavailable or too old; trying Docker..." >&2
if try_docker; then
	exit 0
fi

cat >&2 <<EOF
k6-start: no usable k6 runner found.

Install one of:
  • native: https://grafana.com/docs/k6/latest/set-up/install-k6/
    (macOS: brew install k6)
  • Docker: install Docker and re-run (image ${K6_IMAGE})

Then: make -C test/load k6-start
EOF
exit 1
