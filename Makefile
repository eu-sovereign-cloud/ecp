# ====================================================================================
# Variables
# ====================================================================================
TOOLS_GOMOD := -modfile=./tools/go.mod
GO := go
GO_TOOL := $(GO) run $(TOOLS_GOMOD)

SDK_SRC_DIR := internal/go-sdk/pkg/spec/schema
API_DEST_DIR := apis/generated/types
CRD_TYPES := $(shell (find apis -mindepth 1 -maxdepth 1 -type d | grep -v generated | cut -d'/' -f2))
COMMON_MODELS ?= errors resource
FOUNDATION_SPECS ?= region block-storage
# ====================================================================================

submodules:
	@git submodule sync
	@git submodule update --init --recursive

.PHONY: run-global-server
run-global-server:
	go run ./main.go globalapiserver

.PHONY: generate-crds
generate-crds: $(CRD_TYPES)

.PHONY: $(CRD_TYPES)
$(CRD_TYPES):
	@echo "Generating CRDs for $@"
	@mkdir -p ./apis/generated/crds/$@
	@$(GO_TOOL) -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=".github/boilerplate.go.txt" paths="./apis/$@/v1/..."
	@$(GO_TOOL) -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./apis/$@/v1/... output:crd:artifacts:config=./apis/generated/crds/$@

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
generate-models: generate-commons $(FOUNDATION_SPECS)

.PHONY: $(FOUNDATION_SPECS)
$(FOUNDATION_SPECS):
	@echo "Generating models for $@"
	@GO_TOOL="$(GO_TOOL)" ./scripts/generate-model.sh $@ v1 $(COMMON_MODELS)
	@echo "--------------------------------"

define ECP_MAKE_HELP
ECP Targets:
	generate-common        Generate common models from internal/go-sdk
	generate-models		   Generate models from internal/go-sdk
	generate-crds          Generate CRDs for the regions API
	run-global-server      Run the global API server locally
	create-dev-clusters    Set up one global and one regional cluster for development purposes
	clean-dev-clusters     Remove the global and regional clusters set up for development
	docker-build-images    Build Docker images for the provider components
	generate-regions-crd   Generate CRDs for the regions API from the regions package
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
