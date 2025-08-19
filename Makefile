# ====================================================================================
# Variables
# ====================================================================================
KIND_CLUSTER_NAME=crossplanetest
KIND_VERSION=v0.29.0
CROSSPLANE_NAMESPACE=crossplane-system
TOOLS_GOMOD := -modfile=./tools/go.mod
GO := go
GO_TOOL := $(GO) run $(TOOLS_GOMOD)
crossplane-local-dev: ensure-kind ensure-helm kind-create crossplane-install

ensure-kind:
	@command -v kind >/dev/null 2>&1 || { \
		echo "kind not found, installing..."; \
		curl -Lo ./kind https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-linux-amd64; \
		chmod +x ./kind; \
		sudo mv ./kind /usr/local/bin/kind; \
	}

ensure-helm:
	@command -v helm >/dev/null 2>&1 || { \
		echo "helm not found, installing..."; \
		curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash; \
	}

.PHONY: kind-create
kind-create:
	kind create cluster --name $(KIND_CLUSTER_NAME) || true

.PHONY: crossplane-install
crossplane-install:
	helm repo add crossplane-stable https://charts.crossplane.io/stable
	helm repo update
	helm install crossplane --namespace $(CROSSPLANE_NAMESPACE) --create-namespace crossplane-stable/crossplane

.PHONY: run-global-server
run-global-server:
	go run ./main.go globalapiserver

.PHONY: clean-crossplane-dev
clean-crossplane-dev:
	kubectl delete namespace $(CROSSPLANE_NAMESPACE) || true
	kubectl delete secret example-provider-secret --namespace $(CROSSPLANE_NAMESPACE) || true
	kind delete cluster --name $(KIND_CLUSTER_NAME) || true

.PHONY: generate-crds
generate-crds: generate-regions-crd

# Generate CRDs for the regions API from the regions package.
.PHONY: generate-regions-crd
generate-regions-crd:
	$(GO_TOOL) -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --generate=types -o ./apis/regions/v1/zz_generated_region.go -package v1 ./apis/regions/v1/foundation.region.v1.yaml
	@GO_TOOL="$(GO_TOOL)" ./scripts/prepare-generate-crd.sh \
		./apis/regions/v1/zz_generated_region.go \
		./apis/regions/v1 \
		./apis/generated/regions
	$(GO_TOOL) -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object paths=./apis/regions/v1/; \
	$(GO_TOOL) -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./apis/regions/v1/... output:crd:artifacts:config=./apis/generated/regions

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


define ECP_MAKE_HELP
ECP Targets:
	generate-crds          Generate CRDs for the regions API
	run-global-server      Run the global API server locally
	create-dev-clusters    Set up one global and one regional cluster for development purposes
	clean-dev-clusters     Remove the global and regional clusters set up for development
	docker-build-images    Build Docker images for the provider components
	crossplane-local-dev   Set up a local Crossplane development environment with kind and Helm
	ensure-kind            Ensure kind is installed
	ensure-helm            Ensure Helm is installed
	kind-create            Create a kind cluster for Crossplane development
	crossplane-install     Install Crossplane into the kind cluster
	clean-crossplane-dev   Clean up the Crossplane development environment
	generate-regions-crd   Generate CRDs for the regions API from the regions package
endef

export ECP_MAKE_HELP

.PHONY: help
help:
	@echo "$$ECP_MAKE_HELP"