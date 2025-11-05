# Makefile for api module: generation helpers
# This Makefile provides generate-models, generate-crds and generate-all targets.

# Use the repo tools module by default (relative to foundation/api, tools is at ../tools/go.mod)
TOOLS_GOMOD := -modfile=../tools/go.mod
GO := go
GO_TOOL := $(GO) run $(TOOLS_GOMOD)

.PHONY: generate-all generate-models generate-crds help

generate-all: generate-models generate-crds
	@echo "api: finished generate-all"

generate-models:
	@echo "api: generate-models"
	@GO_TOOL="$(GO_TOOL)" ./scripts/generate-model.sh
	@echo "api: generate-models done"

generate-crds:
	@echo "api: generate-crds"
	@echo "Running go generate (build tag crdgen)..."
	@$(GO) generate -tags=crdgen ./...
	@echo "api: generate-crds done"

help:
	@echo "Targets: generate-models, generate-crds, generate-all"

.PHONY: clean-generated clean-crds
clean-generated:
	@echo "api: cleaning generated files"
	@find generated -type f -name 'zz_generated*' -exec rm -f {} + || true

clean-crds:
	@echo "api: cleaning generated CRD YAMLs"
	@find generated/crds -type f -name '*.yaml' -exec rm -f {} + || true
