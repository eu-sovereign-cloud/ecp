SHELL := /bin/bash

# ====================================================================================
# Kubeconfig and cluster configuration
# ====================================================================================
GLOBAL_KUBECONFIG       ?= ~/.kube/multi-cluster-demo/global-config
REGIONAL_KUBECONFIG     ?= ~/.kube/multi-cluster-demo/regional-config
GLOBAL_CLUSTER_NAME     ?= global
REGIONAL_CLUSTER_NAME   ?= regional

# Plugin image (must match what the plugin deploy Makefile builds)
PLUGIN_IMAGE ?= ionos-plugin:dev

# ====================================================================================
# secatest configuration (override via environment or command line)
# ====================================================================================
SECATEST_REGION              ?= region
SECATEST_TENANT              ?= seca
SECATEST_AUTH_TOKEN          ?= test
SECATEST_SCENARIOS_FILTER    ?= Workspace.V1.ProviderLifeCycle
SECATEST_SCENARIOS_USERS     ?= user1@sdemo.secapi.cloud,user2@demo.secapi.cloud
SECATEST_SCENARIOS_CIDR      ?= 10.1.0.0/16
SECATEST_SCENARIOS_PUBLIC_IPS ?= 52.93.126.1/26
SECATEST_RETRY_BASE_DELAY    ?= 30
SECATEST_RETRY_BASE_INTERVAL ?= 30
SECATEST_RETRY_MAX_ATTEMPTS  ?= 30
SECATEST_RESULTS_PATH        ?= ./results

.PHONY: secatest-scaffolding secatest secatest-all secatest-clean secatest-install-plugin

# ====================================================================================
# Scaffolding — build images, create clusters, deploy gateways + plugin
# ====================================================================================

# Setup the full environment from scratch.
# Usage:
#   make secatest-scaffolding
secatest-scaffolding:
	$(MAKE) -C ../gateway docker-build-images
	$(MAKE) -C ../gateway create-dev-clusters
	$(MAKE) -C ../plugin/ionos/deploy install-all
	$(MAKE) secatest-install-plugin

# Tear down the environment
secatest-clean:
	$(MAKE) -C ../gateway clean-dev-clusters

# Build the ionos plugin Docker image and install it on the regional cluster
secatest-install-plugin:
	$(MAKE) -C ../plugin/ionos/deploy install-plugin-on-regional \
		PLUGIN_IMAGE=$(PLUGIN_IMAGE) \
		REGIONAL_CLUSTER_NAME=$(REGIONAL_CLUSTER_NAME) \
		REGIONAL_KUBECONFIG=$(REGIONAL_KUBECONFIG)

# ====================================================================================
# Run secatest — discovers NodePort endpoints, registers Region CR, runs tests
# ====================================================================================

# Full flow: scaffold + test in one command.
# Usage:
#   make secatest-all
secatest-all: secatest-scaffolding secatest

# Run secatest against in-cluster gateways via NodePort.
#
# Discovers the KIND node IPs and NodePorts for both gateways,
# registers the Region CR with the correct URLs, then runs secatest.
# No port-forwarding needed.
#
# Usage:
#   make secatest
#   make secatest SECATEST_SCENARIOS_FILTER=Compute.V1.ProviderLifeCycle
#   make secatest SECATEST_AUTH_TOKEN=my-token
secatest:
	@GLOBAL_IP=$$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(GLOBAL_CLUSTER_NAME)-control-plane) || { echo "Error: cannot find global cluster node"; exit 1; }; \
	GLOBAL_NP=$$(kubectl --kubeconfig $(GLOBAL_KUBECONFIG) get svc global-api-service -o jsonpath='{.spec.ports[0].nodePort}') || { echo "Error: cannot find global NodePort"; exit 1; }; \
	REGIONAL_IP=$$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(REGIONAL_CLUSTER_NAME)-control-plane) || { echo "Error: cannot find regional cluster node"; exit 1; }; \
	REGIONAL_NP=$$(kubectl --kubeconfig $(REGIONAL_KUBECONFIG) get svc regional-server-svc -o jsonpath='{.spec.ports[0].nodePort}') || { echo "Error: cannot find regional NodePort"; exit 1; }; \
	GLOBAL_URL="http://$${GLOBAL_IP}:$${GLOBAL_NP}"; \
	REGIONAL_URL="http://$${REGIONAL_IP}:$${REGIONAL_NP}"; \
	echo "Global gateway:  $${GLOBAL_URL}"; \
	echo "Regional gateway: $${REGIONAL_URL}"; \
	echo "Waiting for endpoints to be ready..."; \
	for i in $$(seq 1 30); do \
		if curl -s -o /dev/null "$${GLOBAL_URL}" 2>/dev/null && \
		   curl -s -o /dev/null "$${REGIONAL_URL}" 2>/dev/null; then \
			echo "All endpoints ready."; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Error: endpoints did not become ready in time"; exit 1; fi; \
		sleep 1; \
	done; \
	secatest run \
		--provider.region.v1=$${GLOBAL_URL}/providers/seca.region \
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
