# ====================================================================================
# Variables
# ====================================================================================
TOOLS_GOMOD := -modfile=../tools/go.mod
GO := go

# ====================================================================================
submodules:
	@git submodule sync
	@git submodule update --init --recursive
# ====================================================================================
# Generate
# ====================================================================================

.PHONY: generate-all
generate-all: generate-models generate-crds

.PHONY:  generate-models
generate-models:
	@echo "Generating models via api module Makefile"
	@$(MAKE) -C api generate-models
	@echo "--------------------------------"

generate-crds:
	@echo "Generating CRDs via api module Makefile"
	@$(MAKE) -C api generate-crds

# ====================================================================================
# Development
# ====================================================================================
.PHONY: create-dev-clusters
# Sets up one global and one regional cluster for development purposes
create-dev-clusters:
	@echo "Setting up development clusters via gateway Makefile..."
	@$(MAKE) -C gateway create-dev-clusters

.PHONY: run-global-server
run-global-server:
	@echo "Running global API server via gateway Makefile..."
	@$(MAKE) -C gateway run-global-server

.PHONY: run-regional-server
run-regional-server:
	@echo "Running regional API server via gateway Makefile..."
	@$(MAKE) -C gateway run-regional-server

.PHONY: clean-dev-clusters
clean-dev-clusters:
	@echo "Cleaning up development clusters via gateway Makefile..."
	@$(MAKE) -C gateway clean-dev-clusters

.PHONY: docker-build-images
docker-build-images:
	@echo "Building Docker images via gateway Makefile..."
	@$(MAKE) -C gateway docker-build-images

.PHONY: build-gateway
build-gateway:
	@echo "Building gateway binary via gateway Makefile..."
	@$(MAKE) -C gateway build-gateway



define ECP_MAKE_HELP
ECP Targets:
	generate-all		   Generate all code (models and CRDs)
	generate-models		   Generate models from delegator/go-sdk
	generate-crds          Generate CRDs (writes to api/generated/crds)
	run-global-server      Run the global API server locally
	run-regional-server    Run the regional API server locally
	create-dev-clusters    Set up one global and one regional cluster for development purposes
	clean-dev-clusters     Remove the global and regional clusters set up for development
	docker-build-images    Build Docker images for the provider components
	clean				   Clean up generated files
	build-gateway          Build the gateway binary
endef

export ECP_MAKE_HELP

.PHONY: help
help:
	@echo "$$ECP_MAKE_HELP"

.PHONY: clean-generated
clean-generated:
	@echo "Cleaning generated files via api Makefile..."
	@$(MAKE) -C api clean-generated

.PHONY: clean-crds
clean-crds:
	@echo "Cleaning CRDs via api Makefile..."
	@$(MAKE) -C api clean-crds

.PHONY: clean
clean: clean-generated clean-crds
