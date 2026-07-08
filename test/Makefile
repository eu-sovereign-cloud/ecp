# Makefile for the ECP test harness: integration, e2e and conformance suites.

include ../.config.mk

# =====================================================================================
# Variables
# =====================================================================================

# Load environment variables from config.env if it exists
ifneq (,$(wildcard $(CURDIR)/context/config.env))
	include $(CURDIR)/context/config.env
	export
endif

# Image settings
VERSION ?= latest
REGISTRY_URL ?= "default-registry"

# Discover components automatically from the build directory
COMPONENTS := $(notdir $(wildcard $(CURDIR)/build/*))

# Discover deployments automatically from the build directory
DEPLOYMENTS := $(notdir $(wildcard $(CURDIR)/deploy/*))

# Discover components that actually have integration tests
TEST_COMPONENTS := $(notdir $(wildcard $(CURDIR)/integration/*))

# =====================================================================================
# Main Targets
# =====================================================================================
.PHONY: all
all: build-all ## Build all docker images

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_0-9-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# =====================================================================================
# Image Building
# =====================================================================================
.PHONY: build-all
build-all: $(addprefix build-,$(COMPONENTS)) ## Build all component images

define build_rule
.PHONY: build-$(1)
build-$(1): ## Build the docker image for $(1)
	@echo "Building image for $(1)..."
	@DLV_VERSION=$(DLV_VERSION) ./scripts/build.sh $(1)
endef
$(foreach comp,$(COMPONENTS),$(eval $(call build_rule,$(comp))))

# =====================================================================================
# Image Pushing
# =====================================================================================
.PHONY: push-all
push-all: $(addprefix push-,$(COMPONENTS)) ## Push all component images to the registry

define push_rule
.PHONY: push-$(1)
push-$(1): ## Push a specific component image to the registry (e.g., make push-$(1))
	@echo "Pushing image for $(1)..."
	@./scripts/push.sh $(1)
endef
$(foreach comp,$(COMPONENTS),$(eval $(call push_rule,$(comp))))

# =====================================================================================
# Deployment to Kubernetes
# =====================================================================================
.PHONY: deploy-all
deploy-all: $(addprefix deploy-,$(DEPLOYMENTS)) ## Deploy all deployments to the configured Kubernetes cluster

define deploy_rule
.PHONY: deploy-$(1)
deploy-$(1): ## Deploy a specific deployment to the configured Kubernetes cluster (e.g., make deploy-$(1))
	@echo "Deploying $(1) to Kubernetes cluster..."
	@./scripts/deploy.sh $(1)
endef
$(foreach comp,$(DEPLOYMENTS),$(eval $(call deploy_rule,$(comp))))

# =====================================================================================
# KIND Cluster Management
# =====================================================================================
.PHONY: kind-start
kind-start: ## Start a KIND cluster
	@echo "Starting KIND cluster..."
	@./scripts/kind-start.sh

.PHONY: kind-stop
kind-stop: ## Stop the KIND cluster
	@echo "Stopping KIND cluster..."
	@./scripts/kind-stop.sh

.PHONY: kind-load-all
kind-load-all: $(addprefix kind-load-,$(COMPONENTS)) ## Load all component images into the KIND cluster

define kind_load_rule
.PHONY: kind-load-$(1)
kind-load-$(1): ## Load a specific component image into the KIND cluster (e.g., make kind-load-$(1))
	@echo "Loading $(1) image into KIND cluster..."
	@./scripts/kind-load.sh $(1)
endef
$(foreach comp,$(COMPONENTS),$(eval $(call kind_load_rule,$(comp))))

.PHONY: kind-deploy-all
kind-deploy-all: $(addprefix kind-deploy-,$(DEPLOYMENTS)) ## Deploy all deployments to the KIND cluster

define kind_deploy_rule
.PHONY: kind-deploy-$(1)
kind-deploy-$(1): ## Deploy a specific deployment to the KIND cluster (e.g., make kind-deploy-$(1))
	@echo "Deploying $(1) to KIND cluster..."
	@USE_KIND=true ./scripts/deploy.sh $(1)
endef
$(foreach comp,$(DEPLOYMENTS),$(eval $(call kind_deploy_rule,$(comp))))

# --- Test-suite deployment dependencies ----------------------------------------
# Deploying a component that has an integration suite also deploys every other
# component that suite needs, so `make [kind-]deploy-<comp>` is a complete setup for
# `make [kind-]test-<comp>`. Dependencies (see test/integration/<comp>):
#   * test-data        — tenant namespace + storage SKUs + regions (leaf fixture);
#                        the tenant namespace is where workspace/role CRs are created.
#   * delegator        — reconciles CRs to Active (dummy plugin); needs test-data.
#   * gateway-regional — a pure REST↔CR gateway suite; needs only test-data (tenant
#                        namespace + SKUs). It never waits for reconciliation, so it
#                        needs neither the delegator nor the global gateway.
#   * gateway-global   — needs test-data (regions to list, tenant namespace for roles).
# These add-only prerequisite lines augment the recipes generated above; make merges
# duplicate prerequisites, so test-data is applied once per invocation.
deploy-delegator: deploy-test-data
deploy-gateway-regional: deploy-test-data
deploy-gateway-global: deploy-test-data

kind-deploy-delegator: kind-deploy-test-data
kind-deploy-gateway-regional: kind-deploy-test-data
kind-deploy-gateway-global: kind-deploy-test-data

# =====================================================================================
# Cleanup
# =====================================================================================
.PHONY: clean-all
clean-all: ## Clean all deployments and CRDs from the configured Kubernetes cluster
	@$(MAKE) $(addprefix clean-,$(DEPLOYMENTS))
	@$(MAKE) clean-crds

define clean_rule
.PHONY: clean-$(1)
clean-$(1): ## Clean a specific deployment from the configured Kubernetes cluster (e.g., make clean-$(1))
	@echo "Cleaning $(1) from Kubernetes cluster..."
	@./scripts/clean.sh $(1)
endef
$(foreach comp,$(DEPLOYMENTS),$(eval $(call clean_rule,$(comp))))

.PHONY: kind-clean-all
kind-clean-all: ## Clean all deployments and CRDs from the KIND cluster
	@$(MAKE) $(addprefix kind-clean-,$(DEPLOYMENTS))
	@$(MAKE) kind-clean-crds

define kind_clean_rule
.PHONY: kind-clean-$(1)
kind-clean-$(1): ## Clean a specific deployment from the KIND cluster (e.g., make kind-clean-$(1))
	@echo "Cleaning $(1) from KIND cluster..."
	@USE_KIND=true ./scripts/clean.sh $(1)
endef
$(foreach comp,$(DEPLOYMENTS),$(eval $(call kind_clean_rule,$(comp))))

.PHONY: clean-crds
clean-crds: ## Clean all CRDs from the configured Kubernetes cluster
	@echo "Cleaning CRDs..."
	@./scripts/clean-crds.sh

.PHONY: kind-clean-crds
kind-clean-crds: ## Clean all CRDs from the KIND cluster
	@echo "Cleaning CRDs from KIND cluster..."
	@USE_KIND=true ./scripts/clean-crds.sh


# =====================================================================================
# Integration tests — each component (delegator, gateway-global, gateway-regional)
# exercised in isolation. See test/integration/.
# =====================================================================================
.PHONY: integration kind-integration
integration: test-all           ## Run all integration suites against a configured cluster
kind-integration: kind-test-all ## Run all integration suites against a KIND cluster

.PHONY: test-all
test-all: $(addprefix test-,$(TEST_COMPONENTS)) ## Run all integration tests

.PHONY: kind-test-all
kind-test-all: $(addprefix kind-test-,$(TEST_COMPONENTS)) ## Run all integration tests against KIND clusters

# --- Generate test targets for each component ---
define TEST_template
.PHONY: test-$(1)
test-$(1): ## Run $(1) integration tests against a configured cluster
	@./scripts/run-test.sh $(1)

.PHONY: kind-test-$(1)
kind-test-$(1): ## Run $(1) integration tests against a configured KIND cluster
	@USE_KIND=true ./scripts/run-test.sh $(1)
endef

$(foreach comp,$(TEST_COMPONENTS),$(eval $(call TEST_template,$(comp))))

# =====================================================================================
# End-to-end test — the whole stack (both gateways + delegator) in a single run,
# driving the SECA API and asserting reconciliation down to the delegator plugin.
# See test/e2e/. The default plugin is the dummy plugin.
# =====================================================================================

# Which delegator plugin the e2e / conformance stack reconciles with.
E2E_PLUGIN         ?= dummy
CONFORMANCE_PLUGIN ?= dummy

.PHONY: e2e-deploy kind-e2e-deploy
e2e-deploy: ## Deploy the full e2e stack (gateways + delegator) with E2E_PLUGIN (default: dummy)
	@PLUGIN_TYPE=$(E2E_PLUGIN) $(MAKE) deploy-gateway-global deploy-gateway-regional deploy-delegator

kind-e2e-deploy: ## Deploy the full e2e stack into KIND with E2E_PLUGIN (default: dummy)
	@PLUGIN_TYPE=$(E2E_PLUGIN) $(MAKE) kind-deploy-gateway-global kind-deploy-gateway-regional kind-deploy-delegator

.PHONY: e2e kind-e2e
e2e: ## Run the e2e suite against an already-deployed stack (configured cluster)
	@./scripts/run-e2e.sh

kind-e2e: ## One-shot: build, load and deploy the stack into KIND, then run the e2e suite
	@$(MAKE) build-delegator build-gateway-global build-gateway-regional
	@$(MAKE) kind-load-delegator kind-load-gateway-global kind-load-gateway-regional
	@$(MAKE) kind-e2e-deploy
	@USE_KIND=true ./scripts/run-e2e.sh

# =====================================================================================
# Conformance test — runs the SECA conformance suite (secatest) against the stack.
# Generic across plugins: set CONFORMANCE_PLUGIN=aruba to conform-test the aruba
# plugin. Scenarios and client settings are configurable via CONFORMANCE_* vars
# (see scripts/conformance.sh). See test/build/conformance and test/deploy/conformance.
# =====================================================================================
.PHONY: conformance-deploy kind-conformance-deploy
conformance-deploy: ## Deploy the stack for conformance with CONFORMANCE_PLUGIN (default: dummy)
	@PLUGIN_TYPE=$(CONFORMANCE_PLUGIN) $(MAKE) deploy-gateway-global deploy-gateway-regional deploy-delegator

kind-conformance-deploy: ## Deploy the stack for conformance into KIND with CONFORMANCE_PLUGIN
	@PLUGIN_TYPE=$(CONFORMANCE_PLUGIN) $(MAKE) kind-deploy-gateway-global kind-deploy-gateway-regional kind-deploy-delegator

.PHONY: conformance kind-conformance
conformance: ## Run conformance (secatest) against an already-deployed stack (configured cluster)
	@./scripts/conformance.sh

kind-conformance: ## One-shot: build, load and deploy the stack + runner into KIND, then run conformance
	@$(MAKE) build-all
	@$(MAKE) kind-load-all
	@$(MAKE) kind-conformance-deploy
	@USE_KIND=true ./scripts/conformance.sh
