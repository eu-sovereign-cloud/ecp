# ====================================================================================
# Variables
# ====================================================================================
GO := go

# ====================================================================================
# Targets
# ====================================================================================

.PHONY: run-global-server
run-global-server:
	@echo "Running global API server..."
	$(GO) run . global --host=$${HOST:-0.0.0.0} --port=$${PORT:-8080}

.PHONY: run-regional-server
run-regional-server:
	@echo "Running regional API server..."
	$(GO) run . regional --regionalHost=$${REGIONAL_HOST:-0.0.0.0} --regionalPort=$${REGIONAL_PORT:-8080}

.PHONY: docker-build-images
docker-build-images:
	@echo "Building Docker images..."
	@./scripts/build-images.sh

.PHONY: build-gateway
build-gateway:
	@echo "Building gateway binary..."
	@mkdir -p ../bin
	@$(GO) build -o ../bin/gateway .

.PHONY: create-dev-clusters
create-dev-clusters:
	@echo "Setting up development clusters..."
	@./scripts/setup-dev-clusters.sh

.PHONY: clean-dev-clusters
clean-dev-clusters:
	@echo "Executing development cluster cleanup script..."
	@../gateway/scripts/remove-dev-clusters.sh

.PHONY: lint
lint:
	GOWORK=$$(cd ../.. && pwd)/go.work golangci-lint run ./...

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  run-global-server       Run the global API server locally"
	@echo "  run-regional-server     Run the regional API server locally"
	@echo "  docker-build-images     Build Docker images for the gateway"
	@echo "  build-gateway           Build the gateway binary"
	@echo "  create-dev-clusters     Set up development clusters"
	@echo "  lint                    Run linters for the gateway module"
