# Makefile for framework/persistence module: generation helpers
# This Makefile provides generate-models, generate-crds and generate-all targets.

.PHONY: generate-all generate-models generate-crds help

generate-all: generate-models generate-crds
	@echo "framework/persistence: finished generate-all"

generate-models:
	@echo "framework/persistence: generate-models"
	@go run ./cmd/model-gen \
	  --schema-file=../../modules/go-sdk/pkg/spec/schema/resource.go \
	  --output-file=kubernetes/schema/v1/zz_generated_resource.go \
	  --package-name=v1
	@echo "Injecting kubebuilder markers..."
	@go run ./cmd/inject-kubebuilder-markers "./kubernetes/schema/v1"
	@echo "Running controller-gen for DeepCopy on schema/v1..."
	@go tool sigs.k8s.io/controller-tools/cmd/controller-gen object paths="./kubernetes/schema/v1"
	@echo "framework/persistence: generate-models done"

generate-crds:
	@echo "framework/persistence: generate-crds"
	@echo "Running go generate (build tag crdgen)..."
	@go generate -tags=crdgen ./...
	@echo "framework/persistence: generate-crds done"

help:
	@echo "Targets: generate-models, generate-crds, generate-all"

.PHONY: clean-generated clean-crds
clean-generated:
	@echo "framework/persistence: cleaning generated files"
	@rm -f kubernetes/schema/v1/zz_generated_resource.go kubernetes/schema/v1/zz_generated.deepcopy.go

clean-crds:
	@echo "framework/persistence: cleaning generated CRD YAMLs"
	@find kubernetes/crds -type f -name '*.yaml' -exec rm -f {} + 2>/dev/null || true
