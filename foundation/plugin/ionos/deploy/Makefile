SHELL := /bin/bash

# secatest configuration (override via environment or command line)
GLOBAL_KUBECONFIG ?= ~/.kube/multi-cluster-demo/global-config
REGIONAL_KUBECONFIG ?= ~/.kube/multi-cluster-demo/regional-config
GLOBAL_DEPLOYMENT ?= global-api-deployment
REGIONAL_DEPLOYMENT ?= regional-api-deployment
GLOBAL_CLUSTER_PORT ?= 9091
REGIONAL_CLUSTER_PORT ?= 9092
SECATEST_REGION ?= region
SECATEST_TENANT ?= seca
SECATEST_AUTH_TOKEN ?= test
SECATEST_SCENARIOS_FILTER ?= Workspace.V1.LifeCycle
SECATEST_SCENARIOS_USERS ?= user1@sdemo.secapi.cloud,user2@demo.secapi.cloud
SECATEST_SCENARIOS_CIDR ?= 10.1.0.0/16
SECATEST_SCENARIOS_PUBLIC_IPS ?= 52.93.126.1/26
SECATEST_RETRY_BASE_DELAY ?= 30
SECATEST_RETRY_BASE_INTERVAL ?= 30
SECATEST_RETRY_MAX_ATTEMPTS ?= 30
SECATEST_RESULTS_PATH ?= ./results

.PHONY: install-crossplane install-provider create-secret provider-config install-all verify-provider create-token-secret install-on-regional secatest secatest-install-crd secatest-port-forward-global secatest-port-forward-regional

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

# Install the Region CRD on the global cluster
secatest-install-crd:
	kubectl --kubeconfig $(GLOBAL_KUBECONFIG) apply -f secatest-region-crd.yaml

# Port-forward the global cluster (blocking — use standalone for debugging)
secatest-port-forward-global:
	kubectl --kubeconfig $(GLOBAL_KUBECONFIG) port-forward deployment/$(GLOBAL_DEPLOYMENT) $(GLOBAL_CLUSTER_PORT):8080

# Port-forward the regional cluster (blocking — use standalone for debugging)
secatest-port-forward-regional:
	kubectl --kubeconfig $(REGIONAL_KUBECONFIG) port-forward svc/regional-server-svc $(REGIONAL_CLUSTER_PORT):80

# Run secatest e2e tests
# Starts port-forwards for global (:9091) and regional (:9092) clusters in the background,
# waits for them to be ready, runs secatest, then cleans up the port-forwards.
# Usage:
#   make secatest
#   make secatest SECATEST_SCENARIOS_FILTER=Compute.V1.LifeCycle
#   make secatest SECATEST_AUTH_TOKEN=my-token
secatest: secatest-install-crd
	@echo "Starting port-forwards..."
	@kubectl --kubeconfig $(GLOBAL_KUBECONFIG) port-forward deployment/$(GLOBAL_DEPLOYMENT) $(GLOBAL_CLUSTER_PORT):8080 & \
	GLOBAL_PF_PID=$$!; \
	kubectl --kubeconfig $(REGIONAL_KUBECONFIG) port-forward svc/regional-server-svc $(REGIONAL_CLUSTER_PORT):80 & \
	REGIONAL_PF_PID=$$!; \
	trap "echo 'Stopping port-forwards...'; kill $$GLOBAL_PF_PID $$REGIONAL_PF_PID 2>/dev/null || true" EXIT; \
	echo "Waiting for port-forwards to be ready..."; \
	for i in $$(seq 1 30); do \
		if curl -s -o /dev/null http://localhost:$(GLOBAL_CLUSTER_PORT) 2>/dev/null && \
		   curl -s -o /dev/null http://localhost:$(REGIONAL_CLUSTER_PORT) 2>/dev/null; then \
			echo "Port-forwards ready."; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Error: port-forwards did not become ready in time"; exit 1; fi; \
		sleep 1; \
	done; \
	secatest run \
		--provider.region.v1=http://localhost:$(GLOBAL_CLUSTER_PORT)/providers/seca.region \
		--provider.authorization.v1=https://demo.secapi.cloud/providers/seca.authorization \
		--client.tenant=$(SECATEST_TENANT) \
		--client.auth.token="$(SECATEST_AUTH_TOKEN)" \
		--client.region=$(SECATEST_REGION) \
		--scenarios.filter=$(SECATEST_SCENARIOS_FILTER) \
		--scenarios.users=$(SECATEST_SCENARIOS_USERS) \
		--scenarios.cidr=$(SECATEST_SCENARIOS_CIDR) \
		--scenarios.public.ips=$(SECATEST_SCENARIOS_PUBLIC_IPS) \
		--retry.base.delay=$(SECATEST_RETRY_BASE_DELAY) \
		--retry.base.interval=$(SECATEST_RETRY_BASE_INTERVAL) \
		--retry.max.attempts=$(SECATEST_RETRY_MAX_ATTEMPTS) \
		--report.results.path=$(SECATEST_RESULTS_PATH)
