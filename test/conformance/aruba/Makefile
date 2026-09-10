SHELL := /bin/bash

# =====================================================================================
# Aruba real-backend conformance — one KIND cluster, one command.
#
#   make conformance-all \
#       ARUBA_CLIENT_ID=<id> ARUBA_CLIENT_SECRET=<secret> ARUBA_TENANT=<ARU-account>
#
# Creates the cluster, installs the arubacloud-resource-operator with those
# credentials, deploys the ECP stack (both gateways + delegator-aruba + fixtures)
# and runs secatest against the global gateway. Unlike ../ionos this is a single
# cluster: the aruba backend is an operator, not a second control plane, so the
# split topology adds nothing the conformance run exercises.
#
# Everything except the operator install is the harness's own tooling — this
# Makefile only sequences ../../Makefile targets and passes the aruba settings.
# =====================================================================================

TEST_ROOT := $(abspath $(CURDIR)/../..)

# --- Aruba account -------------------------------------------------------------
# The API client the operator authenticates to the Aruba CMP with, and the account
# resources are provisioned in. ARUBA_TENANT is BOTH the SECA tenant the stack is
# seeded for and the tenant secatest drives, so the fixture SKUs land in the same
# hex(sha3-224(tenant)) namespace the plugin reads them from. It must name a REAL
# Aruba account — against anything else nothing ever reaches Active.
ARUBA_CLIENT_ID     ?=
ARUBA_CLIENT_SECRET ?=
ARUBA_TENANT        ?=

# --- Cluster and operator ------------------------------------------------------
KIND_CLUSTER       ?= aruba-conformance
OPERATOR_NAMESPACE ?= aruba-system
OPERATOR_RELEASE   ?= arubacloud-resource-operator
OPERATOR_REPO_NAME ?= arubacloud
OPERATOR_REPO_URL  ?= https://arubacloud.github.io/helm-charts
OPERATOR_VERSION   ?= 1.0.1

# --- secatest ------------------------------------------------------------------
# The stack is deployed with auth on (the harness default), so the runner needs a
# token the dummy authenticator accepts: the "admin" fixture user, whose ra-admin
# assignment is unscoped. CONFORMANCE_REGION must name a Region CR that
# ../../internal/deploy/test-data/regions.yaml publishes.
#
# The retries are far longer than the dummy defaults because every wait here is a
# real CMP provisioning round-trip — a VM boots in minutes, not seconds.
CONFORMANCE_SCENARIOS        ?= .*
CONFORMANCE_REGION           ?= itbg-bergamo
CONFORMANCE_AUTH_TOKEN       ?= $(shell printf '{"username":"admin","password":"e2e-admin-pass"}' | base64 -w0)
CONFORMANCE_RETRY_BASE_DELAY ?= 30
CONFORMANCE_RETRY_BASE_INTERVAL ?= 20
CONFORMANCE_RETRY_MAX_ATTEMPTS  ?= 30

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_0-9-]+:.*?## / {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# =====================================================================================
# The whole flow
# =====================================================================================
.PHONY: conformance-all
conformance-all: scaffolding conformance ## Cluster + operator + stack + secatest, in one command

.PHONY: scaffolding
scaffolding: check-credentials ## Create the KIND cluster, install the operator and deploy the aruba stack
	@kind get clusters 2>/dev/null | grep -qx "$(KIND_CLUSTER)" || \
		$(MAKE) -C $(TEST_ROOT) kind-start KIND_CLUSTER=$(KIND_CLUSTER)
	@$(MAKE) install-operator
	@PLUGIN_TYPE=aruba $(MAKE) -C $(TEST_ROOT) build-all
	@$(MAKE) -C $(TEST_ROOT) kind-load-all KIND_CLUSTER=$(KIND_CLUSTER)
	@$(MAKE) -C $(TEST_ROOT) kind-deploy-stack \
		KIND_CLUSTER=$(KIND_CLUSTER) E2E_PLUGIN=aruba E2E_TENANT=$(ARUBA_TENANT)

# =====================================================================================
# The backend: the third-party operator that actually talks to the Aruba CMP
# =====================================================================================
# Chart: https://github.com/Arubacloud/helm-charts/tree/main/charts/arubacloud-resource-operator
# It brings its own arubacloud.com CRDs (crds.enabled, on by default) — the CRs the
# delegator-aruba plugin writes. Auth mode "single" (the chart default) takes one API
# client for the whole cluster, which is all a conformance run needs; the multi-tenant
# Vault mode is out of scope here.
.PHONY: install-operator
install-operator: check-credentials ## Install the arubacloud-resource-operator with the given credentials
	@helm repo add $(OPERATOR_REPO_NAME) $(OPERATOR_REPO_URL) --force-update
	@helm --kube-context kind-$(KIND_CLUSTER) upgrade --install \
		$(OPERATOR_RELEASE) $(OPERATOR_REPO_NAME)/arubacloud-resource-operator \
		--version $(OPERATOR_VERSION) \
		--namespace $(OPERATOR_NAMESPACE) --create-namespace \
		--set config.auth.mode=single \
		--set config.auth.single.clientId='$(ARUBA_CLIENT_ID)' \
		--set config.auth.single.clientSecret='$(ARUBA_CLIENT_SECRET)' \
		--wait --timeout 5m

# =====================================================================================
# Run secatest against the deployed stack
# =====================================================================================
# conformance.sh is the harness's plugin-generic runner: it renders the runner pod
# and execs secatest in it, targeting the in-cluster global gateway. Only the aruba
# tenant, region and the longer retries are ours.
.PHONY: conformance
conformance: check-tenant ## Run secatest against an already-scaffolded cluster
	@USE_KIND=true KIND_CLUSTER=$(KIND_CLUSTER) \
		CONFORMANCE_TENANT='$(ARUBA_TENANT)' \
		CONFORMANCE_REGION='$(CONFORMANCE_REGION)' \
		CONFORMANCE_SCENARIOS='$(CONFORMANCE_SCENARIOS)' \
		CONFORMANCE_AUTH_TOKEN='$(CONFORMANCE_AUTH_TOKEN)' \
		CONFORMANCE_RETRY_BASE_DELAY='$(CONFORMANCE_RETRY_BASE_DELAY)' \
		CONFORMANCE_RETRY_BASE_INTERVAL='$(CONFORMANCE_RETRY_BASE_INTERVAL)' \
		CONFORMANCE_RETRY_MAX_ATTEMPTS='$(CONFORMANCE_RETRY_MAX_ATTEMPTS)' \
		$(TEST_ROOT)/internal/scripts/conformance.sh

# =====================================================================================
# Teardown
# =====================================================================================
# Deleting the cluster outright strands REAL cloud resources: the SECA CRs' finalizers
# never run, so the plugin never deletes the arubacloud.com CRs and the operator never
# reaps what it provisioned in the CMP. Delete the SECA resources first, in reverse
# dependency order (the order csp/aruba/test/integration tears down in), and only then
# drop the cluster.
SECA_KINDS := instances.compute.v1.secapi.cloud nics.network.v1.secapi.cloud \
	public-ips.network.v1.secapi.cloud subnets.network.v1.secapi.cloud \
	networks.network.v1.secapi.cloud security-groups.network.v1.secapi.cloud \
	security-group-rules.network.v1.secapi.cloud route-tables.network.v1.secapi.cloud \
	internet-gateways.network.v1.secapi.cloud images.storage.v1.secapi.cloud \
	block-storages.storage.v1.secapi.cloud workspaces.workspace.v1.secapi.cloud

.PHONY: clean
clean: ## Delete every provisioned resource, then the KIND cluster
	@for kind in $(SECA_KINDS); do \
		echo "Deleting $${kind}..."; \
		kubectl --context kind-$(KIND_CLUSTER) delete "$${kind}" --all -A --timeout=10m || \
			echo "WARNING: $${kind} teardown did not complete — check the Aruba console for leftovers" >&2; \
	done
	@$(MAKE) -C $(TEST_ROOT) kind-stop KIND_CLUSTER=$(KIND_CLUSTER)

# =====================================================================================
# Guards — fail before standing anything up, not five minutes in
# =====================================================================================
.PHONY: check-credentials check-tenant
check-credentials: check-tenant
	@if [ -z "$(ARUBA_CLIENT_ID)" ] || [ -z "$(ARUBA_CLIENT_SECRET)" ]; then \
		echo "ARUBA_CLIENT_ID and ARUBA_CLIENT_SECRET are required: they are the Aruba API client the operator authenticates with." >&2; \
		exit 1; \
	fi

check-tenant:
	@if [ -z "$(ARUBA_TENANT)" ]; then \
		echo "ARUBA_TENANT is required: it must name a real Aruba account (e.g. ARU-348095), or nothing ever provisions." >&2; \
		exit 1; \
	fi
