#!/usr/bin/env bash
# Run k6 using the runner resolved by k6-start.sh.
#
# Usage:
#   run-k6.sh version
#   run-k6.sh run <script> [extra k6 args...]
#   run-k6.sh run <script> --config <options.json> [...]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOAD_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
CMD_FILE="${SCRIPT_DIR}/.k6-cmd"

if [[ ! -f "${CMD_FILE}" ]]; then
	echo "run-k6: k6 not prepared; run: make -C test/load k6-start" >&2
	exit 1
fi

mode_line="$(cat "${CMD_FILE}")"
mode="${mode_line%%|*}"
target="${mode_line#*|}"

if [[ $# -lt 1 ]]; then
	echo "Usage: $0 version | run <script> [k6-args...]" >&2
	exit 1
fi

action="$1"
shift

# Env vars forwarded into the k6 process (native or docker).
# BASE_URL_* are optional for hello; required by real journeys via lib/config.js.
pass_env=(
	"BASE_URL_GLOBAL=${BASE_URL_GLOBAL:-}"
	"BASE_URL_REGIONAL=${BASE_URL_REGIONAL:-}"
	"E2E_TENANT=${E2E_TENANT:-test-tenant}"
	"AUTH_USER=${AUTH_USER:-admin}"
	"AUTH_PASS=${AUTH_PASS:-e2e-admin-pass}"
	"E2E_AUTH_ENABLED=${E2E_AUTH_ENABLED:-true}"
	"SYSTEM_NAMESPACE=${SYSTEM_NAMESPACE:-e2e-ecp}"
	"WAIT_ACTIVE=${WAIT_ACTIVE:-0}"
	"ACTIVE_TIMEOUT_S=${ACTIVE_TIMEOUT_S:-60}"
	"ACTIVE_POLL_S=${ACTIVE_POLL_S:-2}"
)

run_native() {
	local bin="$1"
	shift
	env "${pass_env[@]}" "${bin}" "$@"
}

run_docker() {
	local image="$1"
	shift
	local docker_env=()
	local e
	for e in "${pass_env[@]}"; do
		docker_env+=(-e "${e}")
	done
	# Mount the load tree so relative imports (lib/, journeys/) resolve.
	docker run --rm -i \
		"${docker_env[@]}" \
		-v "${LOAD_DIR}:/home/k6/load:ro" \
		-w /home/k6/load \
		--network host \
		"${image}" \
		"$@"
}

run_k6() {
	case "${mode}" in
	native) run_native "${target}" "$@" ;;
	docker) run_docker "${target}" "$@" ;;
	*)
		echo "run-k6: unknown mode in ${CMD_FILE}: ${mode}" >&2
		echo "run-k6: re-run make -C test/load k6-start" >&2
		exit 1
		;;
	esac
}

case "${action}" in
version)
	run_k6 version
	;;
run)
	if [[ $# -lt 1 ]]; then
		echo "Usage: $0 run <script> [k6-args...]" >&2
		exit 1
	fi
	script="$1"
	shift
	# Resolve script path relative to LOAD_DIR when not absolute.
	if [[ "${script}" != /* ]]; then
		script="${LOAD_DIR}/${script}"
	fi
	if [[ ! -f "${script}" ]]; then
		echo "run-k6: script not found: ${script}" >&2
		exit 1
	fi
	# For docker, convert host path under LOAD_DIR to container path.
	container_script="${script}"
	if [[ "${mode}" == "docker" ]]; then
		rel="${script#"${LOAD_DIR}"/}"
		if [[ "${rel}" == "${script}" ]]; then
			echo "run-k6: docker mode requires script under ${LOAD_DIR}" >&2
			exit 1
		fi
		container_script="/home/k6/load/${rel}"
	fi
	# Also rewrite --config paths that live under LOAD_DIR for docker.
	args=()
	while [[ $# -gt 0 ]]; do
		case "$1" in
		--config)
			args+=(--config)
			shift
			cfg="${1:-}"
			if [[ -z "${cfg}" ]]; then
				echo "run-k6: --config requires a path" >&2
				exit 1
			fi
			if [[ "${cfg}" != /* ]]; then
				cfg="${LOAD_DIR}/${cfg}"
			fi
			if [[ "${mode}" == "docker" ]]; then
				rel="${cfg#"${LOAD_DIR}"/}"
				if [[ "${rel}" == "${cfg}" ]]; then
					echo "run-k6: docker mode requires --config under ${LOAD_DIR}" >&2
					exit 1
				fi
				cfg="/home/k6/load/${rel}"
			fi
			args+=("${cfg}")
			shift
			;;
		*)
			args+=("$1")
			shift
			;;
		esac
	done
	run_k6 run "${container_script}" "${args[@]+"${args[@]}"}"
	;;
*)
	echo "Usage: $0 version | run <script> [k6-args...]" >&2
	exit 1
	;;
esac
