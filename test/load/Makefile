# k6 functional API load-test harness.

LOAD_DIR := $(CURDIR)
SCRIPTS  := $(LOAD_DIR)/scripts

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
# Public gateway bases for GKE load tests (setup_gke.sh + terraform LBs).
BASE_URL_GLOBAL   ?= http://global.test.ecociel.com
BASE_URL_REGIONAL ?= http://regional.test.ecociel.com
# Helm/gateway namespace for the GKE stack (distinct from KIND's e2e-ecp default).
GKE_SYSTEM_NAMESPACE ?= ecp
# Reports (see README "HTML reports"):
#   REPORT_DASHBOARD=1 (default) — k6 web-dashboard HTML WITH graphs (needs ≥~30s run)
#   REPORT_HTML=0 (default)      — k6-reporter tables/checks (opt-in; no time-series)
REPORT_HTML         ?= 1
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
