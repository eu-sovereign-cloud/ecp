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
	"STEPWISE_PACE_S=${STEPWISE_PACE_S:-1}"
	"STRESS_PACE_S=${STRESS_PACE_S:-1}"
	"THROUGHPUT_RATE=${THROUGHPUT_RATE:-}"
	"THROUGHPUT_DURATION_S=${THROUGHPUT_DURATION_S:-}"
	"THROUGHPUT_PRE_VUS=${THROUGHPUT_PRE_VUS:-}"
	"THROUGHPUT_MAX_VUS=${THROUGHPUT_MAX_VUS:-}"
	"TF_JOURNEY_S=${TF_JOURNEY_S:-300}"
	"TF_DESTROY_BUDGET_S=${TF_DESTROY_BUDGET_S:-45}"
	"TF_RUN_ID=${TF_RUN_ID:-}"
	"TF_ZONE=${TF_ZONE:-itbg-1}"
	"TF_USERS=${TF_USERS:-10}"
	"TF_WORKSPACE_CREATE_PAUSE_S=${TF_WORKSPACE_CREATE_PAUSE_S:-5}"
	"TF_WORKSPACE_WAIT_ATTEMPTS=${TF_WORKSPACE_WAIT_ATTEMPTS:-40}"
	"TF_WORKSPACE_WAIT_PAUSE_S=${TF_WORKSPACE_WAIT_PAUSE_S:-2}"
	"TF_WRITE_RETRIES=${TF_WRITE_RETRIES:-8}"
	"TF_MIN_POLL_BATCH=${TF_MIN_POLL_BATCH:-10}"
	"TF_NS_READY_ATTEMPTS=${TF_NS_READY_ATTEMPTS:-60}"
	"TF_NS_READY_PAUSE_S=${TF_NS_READY_PAUSE_S:-2}"
	"REPORT_HTML=${REPORT_HTML:-0}"
	"K6_HTML_REPORT=${K6_HTML_REPORT:-reports/k6-report.html}"
	"K6_REPORT_TITLE=${K6_REPORT_TITLE:-ECP k6 report}"
	"REPORT_DASHBOARD=${REPORT_DASHBOARD:-1}"
)

# Env flag is truthy (1/true/yes).
env_on() {
	case "${1:-0}" in
	1 | true | TRUE | yes | YES) return 0 ;;
	*) return 1 ;;
	esac
}

report_html_on() { env_on "${REPORT_HTML:-0}"; }
report_dashboard_on() { env_on "${REPORT_DASHBOARD:-0}"; }
report_writes_files() { report_html_on || report_dashboard_on; }

# k6 web dashboard: time-series graphs in a self-contained HTML export.
# Graphs need test duration > 3 * K6_WEB_DASHBOARD_PERIOD (default 10s → ~30s+).
# Port -1 disables the live HTTP server so CI/make does not wait on a browser.
apply_dashboard_env() {
	if ! report_dashboard_on; then
		return 0
	fi
	local export_path="${K6_DASHBOARD_REPORT:-reports/k6-dashboard.html}"
	pass_env+=(
		"K6_WEB_DASHBOARD=true"
		"K6_WEB_DASHBOARD_EXPORT=${export_path}"
		"K6_WEB_DASHBOARD_PERIOD=${K6_WEB_DASHBOARD_PERIOD:-10s}"
		"K6_WEB_DASHBOARD_PORT=${K6_WEB_DASHBOARD_PORT:--1}"
		"K6_WEB_DASHBOARD_OPEN=${K6_WEB_DASHBOARD_OPEN:-false}"
	)
}

ensure_report_dir() {
	if ! report_writes_files; then
		return 0
	fi
	local outs=()
	if report_html_on; then
		outs+=("${K6_HTML_REPORT:-reports/k6-report.html}")
	fi
	if report_dashboard_on; then
		outs+=("${K6_DASHBOARD_REPORT:-reports/k6-dashboard.html}")
	fi
	local out dir
	for out in "${outs[@]}"; do
		if [[ "${out}" != /* ]]; then
			dir="${LOAD_DIR}/$(dirname "${out}")"
		else
			dir="$(dirname "${out}")"
		fi
		mkdir -p "${dir}"
		# Docker mode runs k6 as a fixed non-host uid/gid (e.g. 12345:12345 in
		# grafana/k6), so the mounted reports dir must be world-writable for
		# handleSummary()/dashboard export to succeed.
		if [[ "${mode}" == "docker" ]]; then
			chmod 0777 "${dir}"
		fi
	done
}

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
	# Read-write when reports are on so handleSummary / dashboard export can write.
	local mount_opts="ro"
	if report_writes_files; then
		mount_opts="rw"
	fi
	docker run --rm -i \
		"${docker_env[@]}" \
		-v "${LOAD_DIR}:/home/k6/load:${mount_opts}" \
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
	apply_dashboard_env
	ensure_report_dir
	run_k6 run "${container_script}" "${args[@]+"${args[@]}"}"
	if report_html_on; then
		out="${K6_HTML_REPORT:-reports/k6-report.html}"
		if [[ "${out}" != /* ]]; then
			out="${LOAD_DIR}/${out}"
		fi
		if [[ -f "${out}" ]]; then
			echo "run-k6: k6-reporter (tables/checks): ${out}"
		else
			echo "run-k6: REPORT_HTML set but report not found at ${out}" >&2
		fi
	fi
	if report_dashboard_on; then
		out="${K6_DASHBOARD_REPORT:-reports/k6-dashboard.html}"
		if [[ "${out}" != /* ]]; then
			out="${LOAD_DIR}/${out}"
		fi
		if [[ -f "${out}" ]]; then
			echo "run-k6: web dashboard (graphs): ${out}"
		else
			echo "run-k6: REPORT_DASHBOARD set but export not found at ${out}" >&2
			echo "run-k6: tip: graphs need duration > 3×K6_WEB_DASHBOARD_PERIOD (default 10s → use ≥30s runs; stepwise/stress are ~70s)" >&2
		fi
	fi
	;;
*)
	echo "Usage: $0 version | run <script> [k6-args...]" >&2
	exit 1
	;;
esac
