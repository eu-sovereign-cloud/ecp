# Makefile for persistence module: generation helpers
# This Makefile provides generate-models, generate-crds and generate-all targets.

# Use the repo tools module by default (relative to foundation/persistence, tools is at ../tools/go.mod)
TOOLS_GOMOD := -modfile=../tools/go.mod
GO := go
GO_TOOL := $(GO) run $(TOOLS_GOMOD)

.PHONY: generate-all generate-models generate-crds help

generate-all: generate-models generate-crds
	@echo "persistence: finished generate-all"

generate-models:
	@echo "persistence: generate-models"
	@GO_TOOL="$(GO_TOOL)" ./scripts/generate-model.sh
	@echo "persistence: generate-models done"

generate-crds:
	@echo "persistence: generate-crds"
	@echo "Running go generate (build tag crdgen)..."
	@$(GO) generate -tags=crdgen ./...
	@echo "persistence: generate-crds done"

help:
	@echo "Targets: generate-models, generate-crds, generate-all"

.PHONY: clean-generated clean-crds
clean-generated:
	@echo "persistence: cleaning generated files"
	@find generated -type f -name 'zz_generated*' -exec rm -f {} + || true

clean-crds:
	@echo "persistence: cleaning generated CRD YAMLs"
	@find generated/crds -type f -name '*.yaml' -exec rm -f {} + || true
