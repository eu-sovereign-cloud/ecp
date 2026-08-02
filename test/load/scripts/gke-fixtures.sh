#!/usr/bin/env bash
# Seed cluster fixtures so k6 journeys pass against an external GKE stack
# (setup_gke.sh + terraform LBs). Applies test-data, then rewrites Region
# provider URLs to public BASE_URL_* bases (in-cluster DNS is useless from
# a laptop).
#
# Env:
#   SYSTEM_NAMESPACE   gateway/Helm namespace (default: ecp)
#   E2E_TENANT         fixture tenant (default: test-tenant)
#   BASE_URL_GLOBAL    public global gateway base (default: http://global.test.ecociel.com)
#   BASE_URL_REGIONAL  public regional gateway base (default: http://regional.test.ecociel.com)
#   KUBECONFIG         passed through to kubectl / deploy.sh (optional)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOAD_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEST_DIR="$(cd "${LOAD_DIR}/.." && pwd)"
DEPLOY_SH="${TEST_DIR}/internal/scripts/deploy.sh"

SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-ecp}"
E2E_TENANT="${E2E_TENANT:-test-tenant}"
BASE_URL_GLOBAL="${BASE_URL_GLOBAL:-http://global.test.ecociel.com}"
BASE_URL_REGIONAL="${BASE_URL_REGIONAL:-http://regional.test.ecociel.com}"

# Strip trailing slashes so provider URLs are well-formed.
BASE_URL_GLOBAL="${BASE_URL_GLOBAL%/}"
BASE_URL_REGIONAL="${BASE_URL_REGIONAL%/}"

log() { printf 'gke-fixtures: %s\n' "$*"; }
die() { printf 'gke-fixtures: %s\n' "$*" >&2; exit 1; }

need() {
	command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

need kubectl
need openssl
[[ -x "${DEPLOY_SH}" || -f "${DEPLOY_SH}" ]] || die "deploy.sh not found at ${DEPLOY_SH}"

log "SYSTEM_NAMESPACE=${SYSTEM_NAMESPACE} E2E_TENANT=${E2E_TENANT}"
log "BASE_URL_GLOBAL=${BASE_URL_GLOBAL}"
log "BASE_URL_REGIONAL=${BASE_URL_REGIONAL}"

# 1) Roles, role-assignments, SKUs, sample regions, tenant namespace
log "applying test-data via deploy.sh"
SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE}" E2E_TENANT="${E2E_TENANT}" \
	bash "${DEPLOY_SH}" test-data

# 2) Region matching the GKE regional gateway + public provider URLs
log "applying Region europe-west1 with public regional provider URLs"
kubectl apply -f - <<EOF
apiVersion: v1.secapi.cloud/v1
kind: Region
metadata:
  name: europe-west1
spec:
  availableZones:
    - europe-west1-b
  providers:
    - name: seca.workspace
      url: ${BASE_URL_REGIONAL}/providers/seca.workspace
      version: v1
    - name: seca.storage
      url: ${BASE_URL_REGIONAL}/providers/seca.storage
      version: v1
    - name: seca.network
      url: ${BASE_URL_REGIONAL}/providers/seca.network
      version: v1
    - name: seca.compute
      url: ${BASE_URL_REGIONAL}/providers/seca.compute
      version: v1
EOF

# 3) Rewrite test-data in-cluster Service DNS on other Region CRs so external
#    clients (seca / k6) can use catalog provider URLs too.
if command -v python3 >/dev/null 2>&1; then
	log "rewriting in-cluster region provider URLs to ${BASE_URL_REGIONAL}"
	export BASE_URL_REGIONAL
	# Heredoc is the program; region JSON is passed as argv via process substitution.
	if rewritten="$(python3 - "${BASE_URL_REGIONAL}" <<'PY'
import json, sys, subprocess

base = sys.argv[1].rstrip("/")
try:
    raw = subprocess.check_output(
        ["kubectl", "get", "regions.v1.secapi.cloud", "-o", "json"],
        stderr=subprocess.DEVNULL,
        text=True,
    )
except subprocess.CalledProcessError:
    sys.exit(0)
if not raw.strip():
    sys.exit(0)
data = json.loads(raw)
rewritten = []
for item in data.get("items") or []:
    name = (item.get("metadata") or {}).get("name", "")
    providers = (item.get("spec") or {}).get("providers") or []
    changed = False
    for p in providers:
        url = p.get("url") or ""
        pname = p.get("name") or ""
        if "ecp-regional-gateway-regional" in url or "ecp-gateway-regional" in url:
            if pname:
                p["url"] = f"{base}/providers/{pname}"
                p.setdefault("version", "v1")
                changed = True
    if changed:
        rewritten.append({
            "apiVersion": item.get("apiVersion", "v1.secapi.cloud/v1"),
            "kind": "Region",
            "metadata": {"name": name},
            "spec": item.get("spec", {}),
        })
if not rewritten:
    sys.exit(0)
for i, doc in enumerate(rewritten):
    if i:
        print("---")
    json.dump(doc, sys.stdout)
    print()
PY
	)"; then
		if [[ -n "${rewritten}" ]]; then
			printf '%s\n' "${rewritten}" | kubectl apply -f -
		else
			log "no in-cluster provider URLs to rewrite"
		fi
	else
		log "region URL rewrite skipped (kubectl get regions failed or empty)"
	fi
else
	log "python3 not found; skip in-cluster URL rewrite (europe-west1 still applied)"
fi

# 4) Tenant namespace (no-op if test-data already created it)
log "ensuring tenant namespace"
E2E_TENANT="${E2E_TENANT}" bash "${SCRIPT_DIR}/ensure-tenant.sh"

log "regions now:"
kubectl get regions.v1.secapi.cloud -o custom-columns=NAME:.metadata.name,PROVIDERS:.spec.providers[*].name 2>/dev/null \
	|| kubectl get regions 2>/dev/null \
	|| true

cat <<EOF

gke-fixtures: done.

Export and run smoke:

  export BASE_URL_GLOBAL=${BASE_URL_GLOBAL}
  export BASE_URL_REGIONAL=${BASE_URL_REGIONAL}
  export SYSTEM_NAMESPACE=${SYSTEM_NAMESPACE}
  export E2E_TENANT=${E2E_TENANT}
  export E2E_AUTH_ENABLED=false   # true + AUTH_PASS if auth is on

  make -C test/load smoke
EOF
