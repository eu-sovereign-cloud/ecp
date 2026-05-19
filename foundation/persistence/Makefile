# Makefile for persistence module: generation helpers
# This Makefile provides generate-models, generate-crds and generate-all targets.

.PHONY: generate-all generate-models generate-crds help

generate-all: generate-models generate-crds
	@echo "persistence: finished generate-all"

generate-models:
	@echo "persistence: generate-models"
	@go run ./cmd/model-gen --schema-dir=../../modules/go-sdk/pkg/spec/schema --output-dir=generated/types
	@echo "Injecting kubebuilder markers..."
	@go run ./cmd/inject-kubebuilder-markers "./generated/types"
	@echo "Running controller-gen for DeepCopy..."
	@go tool sigs.k8s.io/controller-tools/cmd/controller-gen object paths="./generated/types"
	@echo "persistence: generate-models done"

generate-crds:
	@echo "persistence: generate-crds"
	@echo "Running go generate (build tag crdgen)..."
	@go generate -tags=crdgen ./...
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
