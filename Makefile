# ====================================================================================
# Variables
# ====================================================================================
TOOLS_GOMOD := -modfile=./tools/go.mod
GO := go
GO_TOOL := $(GO) run
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
	@echo "Generating models into foundation/delegator/api/types"
	@GO_TOOL="$(GO_TOOL)" ./foundation/delegator/scripts/generate-model.sh
	@echo "--------------------------------"

generate-crds:
	@echo "Generating CRDs (output now in foundation/delegator/api/generated/crds) via go generate (build tag crdgen)..."
	@(cd foundation/delegator && go generate -tags=crdgen ./api/...)

# ====================================================================================
# Development
# ====================================================================================
.PHONY: create-dev-clusters
# Sets up one global and one regional cluster for development purposes
create-dev-clusters: docker-build-images
	@echo "Executing development cluster setup script..."
	@./foundation/gateway/scripts/setup-dev-clusters.sh

.PHONY: run-global-server
run-global-server:
	@echo "Running global API server (gateway module)..."
	@(cd foundation/gateway && go run . global --host=$${HOST:-0.0.0.0} --port=$${PORT:-8080})

.PHONY: run-regional-server
run-regional-server:
	@echo "Running regional API server (gateway module)..."
	@(cd foundation/gateway && go run . regional --regionalHost=$${REGIONAL_HOST:-0.0.0.0} --regionalPort=$${REGIONAL_PORT:-8080})

.PHONY: clean-dev-clusters
clean-dev-clusters:
	@echo "Executing development cluster cleanup script..."
	@./foundation/gateway/scripts/remove-dev-clusters.sh

.PHONY: docker-build-images
docker-build-images:
	@echo "Executing image build script..."
	@./foundation/gateway/scripts/build-images.sh

.PHONY: build-gateway
build-gateway:
	@echo "Building gateway binary (foundation module)..."
	@mkdir -p bin
	@(cd foundation/gateway && go build -o ../../bin/gateway .)



define ECP_MAKE_HELP
ECP Targets:
	generate-all		   Generate all code (models and CRDs)
	generate-models		   Generate models from foundation/delegator/go-sdk
	generate-crds          Generate CRDs (writes to foundation/delegator/api/generated/crds)
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
	find foundation/delegator/api/generated -type f -name 'zz_generated*' -exec rm -f {} +

.PHONY: clean-crds
clean-crds:
	find foundation/delegator/api/generated/crds -type f -name '*.yaml' -exec rm -f {} +

.PHONY: clean
clean: clean-generated clean-crds
