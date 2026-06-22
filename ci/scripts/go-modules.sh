#!/usr/bin/env bash
# Print the Go modules subject to repo-wide CI checks (test/lint/vuln/...).
# Derived from the `use (...)` block in go.work, minus modules not subject to
# product CI. If go.work is absent (e.g. parsed inside the builder image build,
# which does not COPY it) prints nothing and exits 0.
#
# Usage: go-modules.sh [--json]
#   (default)  newline-separated module paths
#   --json     compact JSON array, e.g. ["foundation/gateway","foundation/iam"]
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
go_work="${repo_root}/go.work"

# Workspace modules excluded from product CI (tool module, e2e harness).
exclude=" ci/tools/go test/e2e "

modules=()
if [ -r "${go_work}" ]; then
  while IFS= read -r m; do
    [ -n "${m}" ] || continue
    case "${exclude}" in *" ${m} "*) continue ;; esac
    modules+=("${m}")
  done < <(awk '/^use \(/,/^\)/{ if ($1 ~ /^\.\//) print substr($1, 3) }' "${go_work}")
fi

if [ "${1:-}" = "--json" ]; then
  printf '['
  sep=
  for m in ${modules[@]+"${modules[@]}"}; do printf '%s"%s"' "${sep}" "${m}"; sep=,; done
  printf ']\n'
else
  for m in ${modules[@]+"${modules[@]}"}; do printf '%s\n' "${m}"; done
fi
