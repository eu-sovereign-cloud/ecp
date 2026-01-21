# Makefile for Dummy Delegator Plugin

# =====================================================================================
# Variables
# =====================================================================================

# Image settings
REGISTRY ?= "default-registry"
IMG ?= ecp-dummy-delegator
VERSION ?= latest

# K8s settings
KUBECONFIG ?= $(HOME)/.kube/config

# KIND settings
CLUSTER_NAME ?= dummy-delegator-cluster


# =====================================================================================
# Main Targets
# =====================================================================================
.PHONY: all
all: build ## Build the docker image

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_0-9-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)


# =====================================================================================
# Image Building
# =====================================================================================
.PHONY: build
build: ## Build the dummy delegator docker image
	@echo "Building image..."
	@REGISTRY=$(REGISTRY) IMG=$(IMG) VERSION=$(VERSION) ./scripts/build.sh


# =====================================================================================
# Deployment
# =====================================================================================
.PHONY: deploy
deploy: ## Deploy the dummy delegator to the configured Kubernetes cluster
	@echo "Deploying to Kubernetes cluster..."
	@KUBECONFIG=$(KUBECONFIG) REGISTRY=$(REGISTRY) IMG=$(IMG) VERSION=$(VERSION) ./scripts/deploy.sh


# =====================================================================================
# KIND Cluster Management
# =====================================================================================
.PHONY: kind-start
kind-start: ## Start a KIND cluster and deploy the dummy delegator
	@echo "Starting KIND cluster and deploying..."
	@CLUSTER_NAME=$(CLUSTER_NAME) IMG=$(IMG) VERSION=$(VERSION) ./scripts/kind-start.sh

.PHONY: kind-stop
kind-stop: ## Stop the KIND cluster
	@echo "Stopping KIND cluster..."
	@CLUSTER_NAME=$(CLUSTER_NAME) ./scripts/kind-stop.sh

# =====================================================================================
# Integration tests
# =====================================================================================
.PHONY: test-integration
test-integration: ## Run the integration tests
	@echo "Running integration tests..."
	@go test -v -count=1 -tags=integration ./test/integration/...
