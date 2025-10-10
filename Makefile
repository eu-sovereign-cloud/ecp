# ====================================================================================
# Variables
# ====================================================================================
TOOLS_GOMOD := -modfile=./tools/go.mod
GO := go
GO_TOOL := $(GO) run $(TOOLS_GOMOD)

COMMON_MODELS ?= errors resource
FOUNDATION_SPECS ?= region block-storage storage-sku
# ====================================================================================

submodules:
	@git submodule sync
	@git submodule update --init --recursive

.PHONY: run-global-server
run-global-server:
	go run ./main.go globalapiserver

.PHONY: generate-crds
generate-crds:
	@echo "Generating CRDs via go generate (with build tag crdgen)..."
	@$(GO) generate -tags=crdgen ./apis/...

.PHONY: create-dev-clusters
# Sets up one global and one regional cluster for development purposes
create-dev-clusters: docker-build-images
	@echo "Executing development cluster setup script..."
	@./scripts/setup-dev-clusters.sh

.PHONY: clean-dev-clusters
clean-dev-clusters:
	@echo "Executing development cluster cleanup script..."
	@./scripts/remove-dev-clusters.sh

.PHONY: docker-build-images
docker-build-images:
	@echo "Executing image build script..."
	@./scripts/build-images.sh

.PHONY: generate-commons
generate-commons:
	@echo "Generating common models: $(COMMON_MODELS)"
	@GO_TOOL="$(GO_TOOL)" ./scripts/generate-common-models.sh $(COMMON_MODELS)

.PHONY:  generate-models
generate-models: generate-commons $(addprefix model-,$(FOUNDATION_SPECS))

.PHONY: model-%
model-%:
	@echo "Generating models for $*"
	@GO_TOOL="$(GO_TOOL)" ./scripts/generate-model.sh $* v1 $(COMMON_MODELS)
	@echo "--------------------------------"

define ECP_MAKE_HELP
ECP Targets:
	generate-common        Generate common models from internal/go-sdk
	generate-models	   Generate models from internal/go-sdk
	generate-crds          Generate CRDs (regions, block-storage etc.) using controller-gen (uses build tag crdgen)
	run-global-server      Run the global API server locally
	create-dev-clusters    Set up one global and one regional cluster for development purposes
	clean-dev-clusters     Remove the global and regional clusters set up for development
	docker-build-images    Build Docker images for the provider components
endef

export ECP_MAKE_HELP

.PHONY: help
help:
	 @echo "$$ECP_MAKE_HELP"

.PHONY: clean-generated
clean-generated:
	find . -type f -name 'zz_generated*' -exec rm -f {} +

.PHONY: clean-crds
clean-crds:
	find apis/generated/crds -type f -name '*.yaml' -exec rm -f {} +

.PHONY: clean
clean: clean-generated clean-crds
