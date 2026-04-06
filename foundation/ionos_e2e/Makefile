SHELL := /bin/bash

# ====================================================================================
# Kubeconfig and cluster configuration
# ====================================================================================
GLOBAL_KUBECONFIG   ?= ~/.kube/multi-cluster-demo/global-config
REGIONAL_KUBECONFIG ?= ~/.kube/multi-cluster-demo/regional-config
GLOBAL_DEPLOYMENT   ?= global-api-deployment
GLOBAL_CLUSTER_PORT   ?= 9091
REGIONAL_CLUSTER_PORT ?= 9092

# Plugin image (must match what the plugin deploy Makefile builds)
PLUGIN_IMAGE ?= ionos-plugin:dev
# Name of the regional kind cluster (used by `kind load docker-image`)
REGIONAL_CLUSTER_NAME ?= regional

# Gateway ports (used when starting local gateway servers)
GATEWAY_GLOBAL_PORT   ?= 8080
GATEWAY_REGIONAL_PORT ?= 8081

# ====================================================================================
# secatest configuration (override via environment or command line)
# ====================================================================================
SECATEST_REGION             ?= region
SECATEST_TENANT             ?= seca
SECATEST_AUTH_TOKEN         ?= test
SECATEST_SCENARIOS_FILTER   ?= Workspace.V1.ProviderLifeCycle
SECATEST_SCENARIOS_USERS    ?= user1@sdemo.secapi.cloud,user2@demo.secapi.cloud
SECATEST_SCENARIOS_CIDR     ?= 10.1.0.0/16
SECATEST_SCENARIOS_PUBLIC_IPS ?= 52.93.126.1/26
SECATEST_RETRY_BASE_DELAY    ?= 30
SECATEST_RETRY_BASE_INTERVAL ?= 30
SECATEST_RETRY_MAX_ATTEMPTS  ?= 30
SECATEST_RESULTS_PATH        ?= ./results

.PHONY: secatest-scaffolding secatest secatest-install-crd secatest-install-plugin secatest-port-forward-global secatest-port-forward-regional

# Setup the development environment from scratch
# Usage:
#   make secatest-scaffolding IONOS_TOKEN=xxxxx
secatest-scaffolding:
	$(MAKE) -C ../gateway create-dev-clusters
	$(MAKE) -C ../plugin/ionos/deploy install-all
	$(MAKE) secatest-install-plugin
	$(MAKE) secatest-install-crd

secatest-clean-scaffolding:
	$(MAKE) -C ../gateway clean-dev-clusters

# Apply the Region CRD to the global cluster
secatest-install-crd:
	kubectl --kubeconfig $(GLOBAL_KUBECONFIG) apply -f secatest-region-crd.yaml

# Port-forward the global cluster (blocking — use standalone for debugging)
secatest-port-forward-global:
	kubectl --kubeconfig $(GLOBAL_KUBECONFIG) port-forward deployment/$(GLOBAL_DEPLOYMENT) $(GLOBAL_CLUSTER_PORT):8080

# Port-forward the regional cluster (blocking — use standalone for debugging)
secatest-port-forward-regional:
	kubectl --kubeconfig $(REGIONAL_KUBECONFIG) port-forward svc/regional-server-svc $(REGIONAL_CLUSTER_PORT):80

# Build the ionos plugin Docker image and install it on the regional cluster
secatest-install-plugin:
	$(MAKE) -C ../plugin/ionos/deploy install-plugin-on-regional \
		PLUGIN_IMAGE=$(PLUGIN_IMAGE) \
		REGIONAL_CLUSTER_NAME=$(REGIONAL_CLUSTER_NAME) \
		REGIONAL_KUBECONFIG=$(REGIONAL_KUBECONFIG)

# Run secatest e2e tests
#
# Starts the gateway global and regional servers, port-forwards for both clusters,
# waits for all endpoints to be reachable, runs secatest, then cleans up everything.
#
# Usage:
#   make secatest
#   make secatest SECATEST_SCENARIOS_FILTER=Compute.V1.ProviderLifeCycle
#   make secatest SECATEST_AUTH_TOKEN=my-token
secatest: secatest-install-crd secatest-install-plugin
	@echo "Starting gateway global server on port $(GATEWAY_GLOBAL_PORT)..."
	@cd ../gateway && go run . global --port=$(GATEWAY_GLOBAL_PORT) & GATEWAY_GLOBAL_PID=$$!; \
	echo "Starting gateway regional server on port $(GATEWAY_REGIONAL_PORT)..."; \
	cd ../gateway && go run . regional --regionalPort=$(GATEWAY_REGIONAL_PORT) & GATEWAY_REGIONAL_PID=$$!; \
	echo "Starting port-forward for global cluster on port $(GLOBAL_CLUSTER_PORT)..."; \
	kubectl --kubeconfig $(GLOBAL_KUBECONFIG) port-forward deployment/$(GLOBAL_DEPLOYMENT) $(GLOBAL_CLUSTER_PORT):8080 & GLOBAL_PF_PID=$$!; \
	echo "Starting port-forward for regional cluster on port $(REGIONAL_CLUSTER_PORT)..."; \
	kubectl --kubeconfig $(REGIONAL_KUBECONFIG) port-forward svc/regional-server-svc $(REGIONAL_CLUSTER_PORT):80 & REGIONAL_PF_PID=$$!; \
	trap "echo 'Stopping all background processes...'; kill $$GATEWAY_GLOBAL_PID $$GATEWAY_REGIONAL_PID $$GLOBAL_PF_PID $$REGIONAL_PF_PID 2>/dev/null || true" EXIT; \
	echo "Waiting for all endpoints to be ready..."; \
	for i in $$(seq 1 30); do \
		if curl -s -o /dev/null http://localhost:$(GLOBAL_CLUSTER_PORT) 2>/dev/null && \
		   curl -s -o /dev/null http://localhost:$(REGIONAL_CLUSTER_PORT) 2>/dev/null; then \
			echo "All endpoints ready."; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Error: endpoints did not become ready in time"; exit 1; fi; \
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
