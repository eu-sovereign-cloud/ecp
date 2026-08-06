#!/usr/bin/env bash
# Ensure the ECP tenant Namespace exists in the current cluster.
#
# Tenant is not a REST resource: the gateway stores tenant-scoped CRs in a
# Namespace named hex(sha3-224(tenant)), labelled secapi.cloud/tenant-id.
# See test/internal/deploy/test-data/namespace.yaml and deploy.sh.
#
# Env:
#   E2E_TENANT          tenant id (default: test-tenant)
#   CREATE_IF_MISSING   1 (default) create when absent; 0 fail with instructions
#   KUBECONFIG          passed through to kubectl (optional)
set -euo pipefail

TENANT="${E2E_TENANT:-test-tenant}"
CREATE_IF_MISSING="${CREATE_IF_MISSING:-1}"
TENANT_LABEL_KEY="secapi.cloud/tenant-id"

if ! command -v kubectl >/dev/null 2>&1; then
	echo "ensure-tenant: kubectl not found on PATH" >&2
	exit 1
fi
if ! command -v openssl >/dev/null 2>&1; then
	echo "ensure-tenant: openssl not found on PATH (needed for sha3-224)" >&2
	exit 1
fi

NS="$(printf %s "${TENANT}" | openssl dgst -sha3-224 | awk '{print $NF}')"
if [[ -z "${NS}" ]]; then
	echo "ensure-tenant: failed to compute namespace hash for tenant ${TENANT}" >&2
	exit 1
fi

print_manual() {
	cat >&2 <<EOF
Tenant namespace is missing and auto-create is disabled (or failed).

Create it with:

  TENANT=${TENANT}
  NS=\$(printf %s "\$TENANT" | openssl dgst -sha3-224 | awk '{print \$NF}')
  kubectl create namespace "\$NS"
  kubectl label namespace "\$NS" ${TENANT_LABEL_KEY}="\$TENANT"

Or deploy fixtures (preferred for a full stack — includes roles, role-assignments, SKUs):

  make -C test deploy-test-data
  # or: make -C test kind-deploy-test-data

Note: ensure-tenant only creates an empty Namespace. API journeys that need
RBAC still require test-data (or equivalent Role / RoleAssignment CRs).
EOF
}

# Does the namespace exist?
if kubectl get namespace "${NS}" >/dev/null 2>&1; then
	label_val="$(kubectl get namespace "${NS}" -o jsonpath="{.metadata.labels['secapi\.cloud/tenant-id']}" 2>/dev/null || true)"
	if [[ -z "${label_val}" ]]; then
		echo "ensure-tenant: namespace ${NS} exists but has no ${TENANT_LABEL_KEY} label" >&2
		echo "ensure-tenant: refuse to clobber; set the label or pick another E2E_TENANT" >&2
		echo "  kubectl label namespace ${NS} ${TENANT_LABEL_KEY}=${TENANT}" >&2
		exit 1
	fi
	if [[ "${label_val}" != "${TENANT}" ]]; then
		echo "ensure-tenant: namespace ${NS} is labelled ${TENANT_LABEL_KEY}=${label_val}, expected ${TENANT}" >&2
		echo "ensure-tenant: refuse to clobber; fix the label or use a different tenant" >&2
		exit 1
	fi
	echo "ensure-tenant: tenant ${TENANT} ok (namespace ${NS})"
	exit 0
fi

# Missing.
if [[ "${CREATE_IF_MISSING}" != "1" ]]; then
	echo "ensure-tenant: namespace ${NS} for tenant ${TENANT} is missing" >&2
	print_manual
	exit 1
fi

echo "ensure-tenant: creating namespace ${NS} for tenant ${TENANT}..."
if ! kubectl create namespace "${NS}"; then
	echo "ensure-tenant: kubectl create namespace failed" >&2
	print_manual
	exit 1
fi
if ! kubectl label namespace "${NS}" "${TENANT_LABEL_KEY}=${TENANT}"; then
	echo "ensure-tenant: failed to label namespace ${NS}" >&2
	print_manual
	exit 1
fi

echo "ensure-tenant: tenant ${TENANT} created (namespace ${NS})"
echo "ensure-tenant: note — empty Namespace only; deploy test-data for roles/SKUs if needed"
