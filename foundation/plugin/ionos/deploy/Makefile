SHELL := /bin/bash

# Image name and tag for the ionos plugin (override as needed)
PLUGIN_IMAGE ?= ionos-plugin:dev
# Name of the regional kind cluster (used by `kind load docker-image`)
REGIONAL_CLUSTER_NAME ?= regional

.PHONY: install-crossplane install-provider create-secret provider-config install-all verify-provider create-token-secret install-on-regional docker-build-plugin install-plugin-on-regional install-ecp-crds-on-regional

# Crossplane install via Helm
install-crossplane:
	bash ./install-crossplane.sh

# Install IONOS provider. Optional overrides:
# make install-provider PROVIDER_PKG=xpkg.upbound.io/ionos-cloud/provider-upjet-ionoscloud:v0.5.0
# make install-provider PROVIDER_PKG=... PROVIDER_NAME=my-provider
install-provider:
	PROVIDER_PKG=${PROVIDER_PKG} PROVIDER_NAME=${PROVIDER_NAME} bash ./install-provider.sh

# Convenience target to install Crossplane, token secret, and provider on current kube target
install-all: install-crossplane create-token-secret install-provider

# Create a token-based Secret for the provider
create-token-secret:
	IONOS_TOKEN=${IONOS_TOKEN} bash ./create-token-secret.sh

# Create example secret + ProviderConfig (template)
create-secret:
	kubectl apply -f providerconfig-example.yaml

provider-config:
	kubectl apply -f providerconfig-example.yaml

# Verify provider objects and health
verify-provider:
	kubectl get providers.pkg.crossplane.io || true
	kubectl get provider || true
	kubectl get crd | grep -E 'ionos|upjet' || true

# End-to-end install on a regional cluster using a specific kubeconfig
# Usage:
#   make install-on-regional KUBECONFIG=/path/to/regional.kubeconfig IONOS_TOKEN=xxxxx [PROVIDER_PKG=... PROVIDER_NAME=...]
install-on-regional:
	@if [[ -z "$$KUBECONFIG" ]]; then echo "Error: KUBECONFIG must be set"; exit 1; fi
	@if [[ -z "$$IONOS_TOKEN" ]]; then echo "Error: IONOS_TOKEN must be set"; exit 1; fi
	KUBECONFIG=$$KUBECONFIG bash ./install-crossplane.sh
	KUBECONFIG=$$KUBECONFIG IONOS_TOKEN=$$IONOS_TOKEN bash ./create-token-secret.sh
	KUBECONFIG=$$KUBECONFIG PROVIDER_PKG=$${PROVIDER_PKG} PROVIDER_NAME=$${PROVIDER_NAME} bash ./install-provider.sh
	KUBECONFIG=$$KUBECONFIG kubectl apply -f providerconfig-example.yaml

# Build the ionos plugin Docker image
# Build context is the repo root so go.work and local module replaces resolve correctly.
# Usage:
#   make docker-build-plugin
#   make docker-build-plugin PLUGIN_IMAGE=my-registry/ionos-plugin:v1.0
docker-build-plugin:
	docker build \
		-f ../build/Dockerfile \
		-t $(PLUGIN_IMAGE) \
		../../../../

# Install ECP CRDs on the regional cluster (required for the plugin manager to sync caches)
install-ecp-crds-on-regional:
	@if [[ -z "$$REGIONAL_KUBECONFIG" ]]; then echo "Error: REGIONAL_KUBECONFIG must be set"; exit 1; fi
	kubectl --kubeconfig $$REGIONAL_KUBECONFIG apply -f ../../../persistence/generated/crds/workspace/workspace.v1.secapi.cloud_workspaces.yaml
	kubectl --kubeconfig $$REGIONAL_KUBECONFIG apply -f ../../../persistence/generated/crds/storage/storage.v1.secapi.cloud_block-storages.yaml

# Load the plugin image into the regional cluster and apply the Deployment manifest.
# Requires REGIONAL_KUBECONFIG to point at the regional cluster kubeconfig.
# Usage:
#   make install-plugin-on-regional REGIONAL_KUBECONFIG=/path/to/regional.kubeconfig
install-plugin-on-regional: docker-build-plugin install-ecp-crds-on-regional
	@if [[ -z "$$REGIONAL_KUBECONFIG" ]]; then echo "Error: REGIONAL_KUBECONFIG must be set"; exit 1; fi
	kind load docker-image $(PLUGIN_IMAGE) --name $(REGIONAL_CLUSTER_NAME)
	kubectl --kubeconfig $$REGIONAL_KUBECONFIG apply -f rbac.yaml
	kubectl --kubeconfig $$REGIONAL_KUBECONFIG apply -f ionos-plugin-deployment.yaml
	kubectl --kubeconfig $$REGIONAL_KUBECONFIG rollout status deployment/ionos-plugin
