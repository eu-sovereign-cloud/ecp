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
export E2E_TENANT SYSTEM_NAMESPACE AUTH_USER AUTH_PASS E2E_AUTH_ENABLED CREATE_IF_MISSING
export WAIT_ACTIVE

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
