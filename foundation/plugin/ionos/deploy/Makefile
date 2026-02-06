SHELL := /bin/bash

.PHONY: install-crossplane install-provider create-secret provider-config install-all verify-provider create-token-secret install-on-regional

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
