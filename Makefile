# ====================================================================================
# Variables
# ====================================================================================
TOOLS_GOMOD := -modfile=./tools/go.mod
GO := go
GO_TOOL := $(GO) run $(TOOLS_GOMOD)
# ====================================================================================
submodules:
	@git submodule sync
	@git submodule update --init --recursive
# ====================================================================================
# Generate
# ====================================================================================
.PHONY:  generate-models
generate-models:
	@echo "Generating models from go-sdk "
	@GO_TOOL="$(GO_TOOL)" ./scripts/generate-model.sh
	@echo "--------------------------------"

generate-crds:
	@echo "Generating CRDs via go generate (with build tag crdgen)..."
	@$(GO) generate -tags=crdgen ./apis/...

.PHONY: $(CRD_TYPES)
$(CRD_TYPES):
	@echo "Generating CRDs for $@"
	@mkdir -p ./apis/generated/crds/$@
	@$(GO_TOOL) -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=".github/boilerplate.go.txt" paths="./apis/$@/v1/..."
	@$(GO_TOOL) -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./apis/$@/v1/... output:crd:artifacts:config=./apis/generated/crds/$@

# ====================================================================================
# Development
# ====================================================================================
.PHONY: create-dev-clusters
# Sets up one global and one regional cluster for development purposes
create-dev-clusters: docker-build-images
	@echo "Executing development cluster setup script..."
	@./scripts/setup-dev-clusters.sh

.PHONY: run-global-server
run-global-server:
	go run ./main.go globalapiserver

.PHONY: clean-dev-clusters
clean-dev-clusters:
	@echo "Executing development cluster cleanup script..."
	@./scripts/remove-dev-clusters.sh

.PHONY: docker-build-images
docker-build-images:
	@echo "Executing image build script..."
	@./scripts/build-images.sh



define ECP_MAKE_HELP
ECP Targets:
	generate-models		   Generate models from internal/go-sdk
	generate-crds          Generate CRDs based on the crdgen tag
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
