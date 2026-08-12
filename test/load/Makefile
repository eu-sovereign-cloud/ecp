# k6 functional API load-test harness.

LOAD_DIR := $(CURDIR)
SCRIPTS  := $(LOAD_DIR)/scripts

# Local gateway dev clusters (gateway/scripts/setup-dev-clusters.sh): 'global'
# and 'regional' KIND clusters. setup-dev-clusters.sh hardcodes its
# KUBECONFIG_DIR to ~/.kube/multi-cluster-demo, so this must match exactly —
# not independently overridable, or the *-dev targets look in the wrong place.
GATEWAY_SCRIPTS    := $(LOAD_DIR)/../../gateway/scripts
DEV_KUBECONFIG_DIR := $(HOME)/.kube/multi-cluster-demo

# Pin used when falling back to Docker (native k6 may be newer).
K6_IMAGE ?= grafana/k6:0.57.0
export K6_IMAGE

# Optional overrides for later journeys (documented in README).
E2E_TENANT       ?= test-tenant
SYSTEM_NAMESPACE ?= e2e-ecp
AUTH_USER        ?= admin
AUTH_PASS        ?= e2e-admin-pass
E2E_AUTH_ENABLED ?= true
# 1 = create tenant Namespace when missing; 0 = fail with manual instructions.
CREATE_IF_MISSING ?= 1
# 1 = create_workspace polls until status.state is Active (needs delegator).
WAIT_ACTIVE ?= 0
# Public gateway bases (GKE, dev clusters, or local port-forward). No default —
# an empty value here makes each target's own "set BASE_URL_*" guard fire with
# a clear message, instead of a target silently defaulting to a placeholder
# hostname that fails with a confusing DNS lookup error.
BASE_URL_GLOBAL   ?=
BASE_URL_REGIONAL ?=
# Helm/gateway namespace for the GKE stack (distinct from KIND's e2e-ecp default).
GKE_SYSTEM_NAMESPACE ?= ecp
# Reports (see README "HTML reports"):
#   REPORT_DASHBOARD=1 (default) — k6 web-dashboard HTML WITH graphs (needs ≥~30s run)
#   REPORT_HTML=0 (default)      — k6-reporter tables/checks (opt-in; no time-series)
REPORT_HTML         ?= 0
K6_HTML_REPORT      ?= reports/k6-report.html
K6_REPORT_TITLE     ?= ECP k6 report
REPORT_DASHBOARD    ?= 1
K6_DASHBOARD_REPORT ?= reports/k6-dashboard.html
K6_WEB_DASHBOARD_PERIOD ?= 10s
export E2E_TENANT SYSTEM_NAMESPACE AUTH_USER AUTH_PASS E2E_AUTH_ENABLED CREATE_IF_MISSING
export WAIT_ACTIVE BASE_URL_GLOBAL BASE_URL_REGIONAL GKE_SYSTEM_NAMESPACE
export REPORT_HTML K6_HTML_REPORT K6_REPORT_TITLE
export REPORT_DASHBOARD K6_DASHBOARD_REPORT K6_WEB_DASHBOARD_PERIOD

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_0-9-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: k6-start
k6-start: ## Ensure k6 is available (native binary preferred, Docker fallback)
	@$(SCRIPTS)/k6-start.sh

.PHONY: k6-version
k6-version: k6-start ## Print the resolved k6 version
	@$(SCRIPTS)/run-k6.sh version

.PHONY: hello
hello: k6-start ## Run a no-network k6 self-check (no BASE_URL required)
	@$(SCRIPTS)/run-k6.sh run journeys/hello.js

.PHONY: selfcheck
selfcheck: k6-start ## Offline lib check (config/auth) with smoke options
	@$(SCRIPTS)/run-k6.sh run journeys/_selfcheck.js --config options/smoke.json

.PHONY: ensure-tenant
ensure-tenant: ## Ensure tenant Namespace exists (kubectl; create if missing by default)
	@$(SCRIPTS)/ensure-tenant.sh

.PHONY: gke-fixtures
gke-fixtures: ## Seed GKE fixtures (test-data + public Region URLs) for load journeys
	@SYSTEM_NAMESPACE="$(GKE_SYSTEM_NAMESPACE)" \
		E2E_TENANT="$(E2E_TENANT)" \
		BASE_URL_GLOBAL="$(BASE_URL_GLOBAL)" \
		BASE_URL_REGIONAL="$(BASE_URL_REGIONAL)" \
		$(SCRIPTS)/gke-fixtures.sh

.PHONY: smoke
smoke: k6-start ## Run smoke journey (healthz, regions, list workspaces)
	@if [ -z "$(BASE_URL_GLOBAL)" ] || [ -z "$(BASE_URL_REGIONAL)" ]; then \
		echo "smoke: set BASE_URL_GLOBAL and BASE_URL_REGIONAL (see test/load/README.md)" >&2; \
		exit 1; \
	fi
	@$(MAKE) ensure-tenant
	@$(SCRIPTS)/run-k6.sh run journeys/smoke.js --config options/smoke.json

.PHONY: ensure-dev-clusters
ensure-dev-clusters: ## Create local 'global'/'regional' gateway dev clusters if missing (build images + create-dev-clusters)
	@$(GATEWAY_SCRIPTS)/ensure-dev-clusters.sh

# Discover BASE_URL_GLOBAL/REGIONAL from the dev clusters' NodePorts, point
# KUBECONFIG at the regional cluster (ensure-tenant and journeys' tenant CRs
# live there, not on the host's ambient/default context), and use tenant
# "seca" — the tenant setup-dev-clusters.sh's regional fixtures actually seed
# (namespace + StorageSKU), not the GKE/e2e-cluster default "test-tenant".
# Re-invoke $(MAKE) with these as environment overrides (env wins over the
# `?=` defaults above).
DEV_ENV = KUBECONFIG=$(DEV_KUBECONFIG_DIR)/regional-config \
	E2E_TENANT=seca \
	BASE_URL_GLOBAL=$$($(GATEWAY_SCRIPTS)/dev-url.sh global "$(DEV_KUBECONFIG_DIR)/global-config" global-api-service) \
	BASE_URL_REGIONAL=$$($(GATEWAY_SCRIPTS)/dev-url.sh regional "$(DEV_KUBECONFIG_DIR)/regional-config" regional-server-svc)

.PHONY: smoke-dev
smoke-dev: k6-start ensure-dev-clusters ## Run smoke against local 'global'/'regional' gateway dev clusters (auto-creates clusters + discovers BASE_URL_*, no port-forward)
	@$(DEV_ENV) $(MAKE) smoke

.PHONY: create-workspace-dev
create-workspace-dev: k6-start ensure-dev-clusters ## create-workspace against local gateway dev clusters (auto-creates clusters + discovers BASE_URL_*)
	@$(DEV_ENV) $(MAKE) create-workspace

.PHONY: stepwise-dev
stepwise-dev: k6-start ensure-dev-clusters ## stepwise against local gateway dev clusters (auto-creates clusters + discovers BASE_URL_*)
	@$(DEV_ENV) $(MAKE) stepwise

.PHONY: stress-dev
stress-dev: k6-start ensure-dev-clusters ## stress against local gateway dev clusters (auto-creates clusters + discovers BASE_URL_*)
	@$(DEV_ENV) $(MAKE) stress

.PHONY: tf-stress-dev
tf-stress-dev: k6-start ensure-dev-clusters ## tf-stress against local gateway dev clusters (auto-creates clusters + discovers BASE_URL_*)
	@$(DEV_ENV) $(MAKE) tf-stress

.PHONY: tf-net-storage-dev
tf-net-storage-dev: k6-start ensure-dev-clusters ## tf-net-storage against local gateway dev clusters (auto-creates clusters + discovers BASE_URL_*)
	@$(DEV_ENV) $(MAKE) tf-net-storage

.PHONY: create-workspace
create-workspace: k6-start ## Create/get/delete a workspace via regional API
	@if [ -z "$(BASE_URL_GLOBAL)" ] || [ -z "$(BASE_URL_REGIONAL)" ]; then \
		echo "create-workspace: set BASE_URL_GLOBAL and BASE_URL_REGIONAL (see test/load/README.md)" >&2; \
		exit 1; \
	fi
	@$(MAKE) ensure-tenant
	@$(SCRIPTS)/run-k6.sh run journeys/create_workspace.js --config options/smoke.json

.PHONY: stepwise
stepwise: k6-start ## Stepwise load: 1 VU warmup then 6×10s phases VUs 2..12
	@if [ -z "$(BASE_URL_GLOBAL)" ] || [ -z "$(BASE_URL_REGIONAL)" ]; then \
		echo "stepwise: set BASE_URL_GLOBAL and BASE_URL_REGIONAL (see test/load/README.md)" >&2; \
		exit 1; \
	fi
	@$(MAKE) ensure-tenant
	@$(SCRIPTS)/run-k6.sh run journeys/stepwise.js --config options/stepwise.json

.PHONY: stress
stress: k6-start ## Stress load: 1 VU warmup then 12×5s phases VUs 30..360
	@if [ -z "$(BASE_URL_GLOBAL)" ] || [ -z "$(BASE_URL_REGIONAL)" ]; then \
		echo "stress: set BASE_URL_GLOBAL and BASE_URL_REGIONAL (see test/load/README.md)" >&2; \
		exit 1; \
	fi
	@$(MAKE) ensure-tenant
	@$(SCRIPTS)/run-k6.sh run journeys/stress.js --config options/stress.json

.PHONY: throughput
throughput: k6-start ## Sustained-rate check (constant-arrival-rate; dropped_iterations + per-op duration SLOs)
	@if [ -z "$(BASE_URL_GLOBAL)" ] || [ -z "$(BASE_URL_REGIONAL)" ]; then \
		echo "throughput: set BASE_URL_GLOBAL and BASE_URL_REGIONAL (see test/load/README.md)" >&2; \
		exit 1; \
	fi
	@$(MAKE) ensure-tenant
	@K6_DASHBOARD_REPORT=reports/throughput-dashboard.html \
		K6_REPORT_TITLE='ECP throughput' \
		K6_HTML_REPORT=reports/throughput-report.html \
		$(SCRIPTS)/run-k6.sh run journeys/throughput.js

.PHONY: throughput-dev
throughput-dev: k6-start ensure-dev-clusters ## throughput against local gateway dev clusters (auto-creates clusters + discovers BASE_URL_*)
	@$(DEV_ENV) $(MAKE) throughput

.PHONY: tf-stress
tf-stress: k6-start ## Terraform-like apply ×10 workspaces (~5 min, fixed polls, destroy)
	@if [ -z "$(BASE_URL_GLOBAL)" ] || [ -z "$(BASE_URL_REGIONAL)" ]; then \
		echo "tf-stress: set BASE_URL_GLOBAL and BASE_URL_REGIONAL (see test/load/README.md)" >&2; \
		exit 1; \
	fi
	@$(MAKE) ensure-tenant
	@K6_DASHBOARD_REPORT=reports/tf-stress-dashboard.html \
		K6_REPORT_TITLE='ECP tf-stress' \
		K6_HTML_REPORT=reports/tf-stress-report.html \
		$(SCRIPTS)/run-k6.sh run journeys/tf_stress.js --config options/tf-stress.json

.PHONY: tf-net-storage
tf-net-storage: k6-start ## TF-like: slow workspace bootstrap, then network+block-storage only
	@if [ -z "$(BASE_URL_GLOBAL)" ] || [ -z "$(BASE_URL_REGIONAL)" ]; then \
		echo "tf-net-storage: set BASE_URL_GLOBAL and BASE_URL_REGIONAL (see test/load/README.md)" >&2; \
		exit 1; \
	fi
	@$(MAKE) ensure-tenant
	@K6_DASHBOARD_REPORT=reports/tf-net-storage-dashboard.html \
		K6_REPORT_TITLE='ECP tf-net-storage' \
		K6_HTML_REPORT=reports/tf-net-storage-report.html \
		$(SCRIPTS)/run-k6.sh run journeys/tf_net_storage.js --config options/tf-net-storage.json

.PHONY: apply-dashboards
apply-dashboards: ## Apply Grafana dashboards to monitoring ConfigMap
	@$(LOAD_DIR)/observability/apply-dashboards.sh
