# Makefile for the Aruba Delegator Plugin

include ../../.config.mk

# =====================================================================================
# Main Targets
# =====================================================================================
.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_0-9-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# =====================================================================================
# Integration tests
# =====================================================================================
# The suite drives the plugin directly (SECA CRs, no gateway) and asserts the delegator reconciles
# them into arubacloud.com CRs that the operator provisions. Unlike the dummy suite it does NOT stand
# up a cluster: the arubacloud-resource-operator + Aruba credentials must be installed out of band and
# the delegator-aruba deployed (the two-phase flow in ../../test/conformance/aruba). A REAL Aruba
# account is required for resources to reach Active.
#
#   make test-integration ARUBA_TENANT=ARU-348095
#
.PHONY: test-integration
test-integration: ## Run the aruba plugin integration suite (needs a real cluster + Aruba account)
	@echo "Running aruba integration tests (tenant: $(or $(ARUBA_TENANT),test-tenant))..."
	@go test -v -count=1 -tags=integration -timeout 40m ./test/integration/...
