# Makefile for the ECP test harness: integration, e2e and conformance suites.

include ../.config.mk

# =====================================================================================
# Variables
# =====================================================================================

# Load environment variables from config.env if it exists
ifneq (,$(wildcard $(CURDIR)/internal/context/config.env))
	include $(CURDIR)/internal/context/config.env
	export
	# USE_KIND must come only from KIND-specific recipe prefixes; never let a
	# stray config.env value leak into every target via the blanket export above.
	unexport USE_KIND
endif

# Image settings
VERSION ?= latest
REGISTRY_URL ?= "default-registry"

# --- e2e / deploy customization --------------------------------------------------
# Override to target a custom deployment. Threaded (via export below) to deploy.sh
# and the e2e suite, so pass the SAME values to *-deploy and e2e:
#   make e2e-deploy SYSTEM_NAMESPACE=my-ecp E2E_TENANT=my-tenant
#   make e2e        SYSTEM_NAMESPACE=my-ecp E2E_TENANT=my-tenant
# CLUSTER: export KUBECONFIG (or drop internal/context/kubeconfig.yaml) to pick it.
#   SYSTEM_NAMESPACE — namespace the components (gateways + delegator) deploy into.
#   E2E_TENANT       — fixture tenant; its data lands in hex(sha3-224(tenant)).
SYSTEM_NAMESPACE ?= e2e-ecp
E2E_TENANT       ?= test-tenant
export SYSTEM_NAMESPACE E2E_TENANT

# Discover components automatically from the build directory
COMPONENTS := $(notdir $(wildcard $(CURDIR)/internal/build/*))

# Discover deployments automatically from the deploy directory
DEPLOYMENTS := $(notdir $(wildcard $(CURDIR)/internal/deploy/*))

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
	@DLV_VERSION=$(DLV_VERSION) ./internal/scripts/build.sh $(1)
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
	@./internal/scripts/push.sh $(1)
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
	@./internal/scripts/deploy.sh $(1)
endef
$(foreach comp,$(DEPLOYMENTS),$(eval $(call deploy_rule,$(comp))))

# =====================================================================================
# KIND Cluster Management
# =====================================================================================
.PHONY: kind-start
kind-start: ## Start a KIND cluster
	@echo "Starting KIND cluster..."
	@./internal/scripts/kind-start.sh

.PHONY: kind-stop
kind-stop: ## Stop the KIND cluster
	@echo "Stopping KIND cluster..."
	@./internal/scripts/kind-stop.sh

.PHONY: kind-load-all
kind-load-all: $(addprefix kind-load-,$(COMPONENTS)) ## Load all component images into the KIND cluster

define kind_load_rule
.PHONY: kind-load-$(1)
kind-load-$(1): ## Load a specific component image into the KIND cluster (e.g., make kind-load-$(1))
	@echo "Loading $(1) image into KIND cluster..."
	@./internal/scripts/kind-load.sh $(1)
endef
$(foreach comp,$(COMPONENTS),$(eval $(call kind_load_rule,$(comp))))

.PHONY: kind-deploy-all
kind-deploy-all: $(addprefix kind-deploy-,$(DEPLOYMENTS)) ## Deploy all deployments to the KIND cluster

define kind_deploy_rule
.PHONY: kind-deploy-$(1)
kind-deploy-$(1): ## Deploy a specific deployment to the KIND cluster (e.g., make kind-deploy-$(1))
	@echo "Deploying $(1) to KIND cluster..."
	@USE_KIND=true ./internal/scripts/deploy.sh $(1)
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
	@./internal/scripts/clean.sh $(1)
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
	@USE_KIND=true ./internal/scripts/clean.sh $(1)
endef
$(foreach comp,$(DEPLOYMENTS),$(eval $(call kind_clean_rule,$(comp))))

.PHONY: clean-crds
clean-crds: ## Clean all CRDs from the configured Kubernetes cluster
	@echo "Cleaning CRDs..."
	@./internal/scripts/clean-crds.sh

.PHONY: kind-clean-crds
kind-clean-crds: ## Clean all CRDs from the KIND cluster
	@echo "Cleaning CRDs from KIND cluster..."
	@USE_KIND=true ./internal/scripts/clean-crds.sh


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
	@./internal/scripts/run-test.sh $(1)

.PHONY: kind-test-$(1)
kind-test-$(1): ## Run $(1) integration tests against a configured KIND cluster
	@USE_KIND=true ./internal/scripts/run-test.sh $(1)
endef

$(foreach comp,$(TEST_COMPONENTS),$(eval $(call TEST_template,$(comp))))

# =====================================================================================
# End-to-end test — the whole stack (both gateways + delegator) in a single run,
# driving the SECA API and asserting reconciliation down to the delegator plugin.
# See test/e2e/. The default plugin is the dummy plugin.
# =====================================================================================

# Which delegator plugin the e2e / conformance stack reconciles with. Only the
# dummy plugin is self-contained (no external backend), so the one-shot
# `kind-e2e` / `kind-conformance` targets always use dummy. The aruba and ionos
# plugins need their backend stood up first, so they are two-phase — set these on
# the `*-deploy` targets, provision the backend, then run `e2e` / `conformance`:
#   make conformance-deploy CONFORMANCE_PLUGIN=aruba   # deploy stack + aruba plugin
#   <install the aruba operator + credentials>
#   make conformance                                   # run secatest
# For a full real-backend IONOS run see conformance-ionos* (conformance/ionos/).
E2E_PLUGIN         ?= dummy
CONFORMANCE_PLUGIN ?= dummy

# The e2e stack deploys its GLOBAL gateway with the jwt authenticator and leaves
# the regional one on dummy, so one e2e run covers both: the suite signs real
# JWTs for the global gateway and dummy tokens for the regional one. The
# integration suites deploy the same gateway without AUTH_PLUGIN and get dummy.
# The e2e suite assumes this split, so pair `e2e` with `e2e-deploy`.
.PHONY: e2e-deploy kind-e2e-deploy
e2e-deploy: ## Deploy the full e2e stack (gateways + delegator) with E2E_PLUGIN (default: dummy)
	@AUTH_PLUGIN=jwt $(MAKE) deploy-gateway-global
	@PLUGIN_TYPE=$(E2E_PLUGIN) $(MAKE) deploy-gateway-regional deploy-delegator

kind-e2e-deploy: ## Deploy the full e2e stack into KIND with E2E_PLUGIN (default: dummy)
	@AUTH_PLUGIN=jwt $(MAKE) kind-deploy-gateway-global
	@PLUGIN_TYPE=$(E2E_PLUGIN) $(MAKE) kind-deploy-gateway-regional kind-deploy-delegator

.PHONY: e2e kind-e2e
e2e: ## Run the e2e suite against an already-deployed stack (configured cluster)
	@./internal/scripts/run-e2e.sh

kind-e2e: ## One-shot (dummy plugin only): build, load, deploy the stack into KIND and run the e2e suite
	@$(MAKE) build-delegator build-gateway-global build-gateway-regional
	@$(MAKE) kind-load-delegator kind-load-gateway-global kind-load-gateway-regional
	@$(MAKE) kind-e2e-deploy E2E_PLUGIN=dummy
	@USE_KIND=true ./internal/scripts/run-e2e.sh

# =====================================================================================
# Conformance test — runs the SECA conformance suite (secatest) against the stack.
# Generic across plugins: set CONFORMANCE_PLUGIN=aruba to conform-test the aruba
# plugin. Scenarios and client settings are configurable via CONFORMANCE_* vars
# (see internal/scripts/conformance.sh, internal/build/conformance, internal/deploy/conformance).
# =====================================================================================
.PHONY: conformance-deploy kind-conformance-deploy
conformance-deploy: ## Deploy the stack for conformance with CONFORMANCE_PLUGIN (default: dummy)
	@PLUGIN_TYPE=$(CONFORMANCE_PLUGIN) $(MAKE) deploy-gateway-global deploy-gateway-regional deploy-delegator

kind-conformance-deploy: ## Deploy the stack for conformance into KIND with CONFORMANCE_PLUGIN
	@PLUGIN_TYPE=$(CONFORMANCE_PLUGIN) $(MAKE) kind-deploy-gateway-global kind-deploy-gateway-regional kind-deploy-delegator

.PHONY: conformance kind-conformance
conformance: ## Run conformance (secatest) against an already-deployed stack (configured cluster)
	@./internal/scripts/conformance.sh

kind-conformance: ## One-shot (dummy plugin only): build, load, deploy stack + runner into KIND and run conformance
	@$(MAKE) build-all
	@$(MAKE) kind-load-all
	@$(MAKE) kind-conformance-deploy CONFORMANCE_PLUGIN=dummy
	@USE_KIND=true ./internal/scripts/conformance.sh

# --- IONOS conformance (real backend) ----------------------------------------
# The IONOS plugin can run in the delegator (CONFORMANCE_PLUGIN=ionos), but that
# only loads the controllers. This flow is the full real-backend run: it stands up
# the IONOS multi-cluster demo, installs Crossplane + the IONOS provider, and runs
# secatest against it. It lives standalone in conformance/ionos/ and is delegated
# to here so it is reachable from this single Makefile.
.PHONY: conformance-ionos conformance-ionos-scaffolding conformance-ionos-clean
conformance-ionos: ## Run the IONOS secatest conformance against the multi-cluster demo
	@$(MAKE) -C conformance/ionos secatest

conformance-ionos-scaffolding: ## Build images, create the demo clusters and install the IONOS plugin
	@$(MAKE) -C conformance/ionos secatest-scaffolding

conformance-ionos-clean: ## Tear down the IONOS conformance demo clusters
	@$(MAKE) -C conformance/ionos secatest-clean
# Benchmark & report
# =====================================================================================
# bench — fire the load workload against the deployed gateway to populate metrics.
# Requires a running cluster with auth enabled (the e2e default).
#
#   make bench                        # 500 requests (default)
#   E2E_BENCH_REQUESTS=1000 make bench
#   USE_KIND=true make bench
#
.PHONY: bench
bench: ## Run the benchmark load workload against the deployed gateway (E2E_BENCH=1)
	@E2E_BENCH=1 ./internal/scripts/run-test.sh gateway-global -run=TestBench

.PHONY: kind-bench
kind-bench: ## Run the benchmark load workload against the KIND gateway (E2E_BENCH=1)
	@E2E_BENCH=1 USE_KIND=true ./internal/scripts/run-test.sh gateway-global -run=TestBench

# report — scrape /metrics from the deployed gateway and emit report/REPORT.md.
# Set SNAP_FILE to save the raw snapshot; set IMPL_TAG to label the series.
#
#   make report                                  # snap to report/snap.txt, tag=default
#   IMPL_TAG=cached make report                  # tag the series as "cached"
#   SNAP_FILE=report/snap-direct.txt make report # save to a named file
#
.PHONY: report
report: ## Scrape gateway /metrics and generate report/REPORT.md via benchreport
	@./internal/scripts/report.sh
