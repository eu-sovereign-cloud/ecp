# Makefile for framework/backend/kubernetes module: generation helpers
# This Makefile provides generate-models, generate-crds and generate-all targets.

# Repo root, derived from this Makefile's location (framework/backend/kubernetes/).
REPO_ROOT := $(abspath $(CURDIR)/../../..)

# All CRD YAML files are written exclusively to chart/crd/ at the repo root.
CRD_OUT_DIR := $(REPO_ROOT)/chart/crd

.PHONY: generate-all generate-models generate-crds help

generate-all: generate-models generate-crds
	@echo "framework/backend/kubernetes: finished generate-all"

generate-models:
	@echo "framework/backend/kubernetes: generate-models"
	@go run ./cmd/model-gen \
	  --schema-file=../../../modules/go-sdk/pkg/spec/schema/resource.go \
	  --output-file=schema/v1/zz_generated_resource.go \
	  --package-name=v1
	@echo "Injecting kubebuilder markers..."
	@go run ./cmd/inject-kubebuilder-markers "./schema/v1"
	@echo "Running controller-gen for DeepCopy on schema/v1..."
	@go tool sigs.k8s.io/controller-tools/cmd/controller-gen object paths="./schema/v1"
	@echo "framework/backend/kubernetes: generate-models done"

# generate-crds: inject +kubebuilder:validation:* markers into every resource-slice
# zz_generated_schema.go (the inject tool is idempotent), then run controller-gen
# crd over all slices to emit 18 CRD YAMLs into chart/crd/ at the repo root.
#
# The inject tool lives in this module (./cmd/inject-kubebuilder-markers); it is
# built once and reused across all 18 slice directories. controller-gen runs via
# go tool (this module's tool dependency at v0.20.0); the workspace (go.work) is
# automatically active, allowing cross-module package resolution so that the
# schemav1.CommonData embed from framework resolves correctly.
generate-crds:
	@echo "framework/backend/kubernetes: generate-crds"
	@echo "Building inject-kubebuilder-markers tool..."
	@go build -o /tmp/ecp-inject-kubebuilder-markers ./cmd/inject-kubebuilder-markers
	@echo "Injecting kubebuilder markers into resource slices..."
	@for dir in \
	    "$(REPO_ROOT)/resource/authorization/v1/role/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/authorization/v1/role-assignment/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/compute/v1/instance/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/compute/v1/sku/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/network/v1/internet-gateway/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/network/v1/network-sku/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/network/v1/network/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/network/v1/nic/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/network/v1/public-ip/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/network/v1/route-table/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/network/v1/security-group-rule/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/network/v1/security-group/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/network/v1/subnet/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/region/v1/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/storage/v1/block-storage/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/storage/v1/image/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/storage/v1/storage-sku/backend/kubernetes" \
	    "$(REPO_ROOT)/resource/workspace/v1/backend/kubernetes"; do \
	    /tmp/ecp-inject-kubebuilder-markers "$$dir"; \
	done
	@rm -f /tmp/ecp-inject-kubebuilder-markers
	@echo "Running controller-gen for CRD YAML..."
	@mkdir -p $(CRD_OUT_DIR)
	@go tool sigs.k8s.io/controller-tools/cmd/controller-gen crd \
	    paths="github.com/eu-sovereign-cloud/ecp/resource/..." \
	    output:crd:dir=$(CRD_OUT_DIR)
	@echo "framework/backend/kubernetes: generate-crds done"

help:
	@echo "Targets: generate-models, generate-crds, generate-all"

.PHONY: clean-generated clean-crds
clean-generated:
	@echo "framework/backend/kubernetes: cleaning generated files"
	@rm -f schema/v1/zz_generated_resource.go schema/v1/zz_generated.deepcopy.go

clean-crds:
	@echo "framework/backend/kubernetes: cleaning generated CRD YAMLs"
	@find $(CRD_OUT_DIR) -type f -name '*.yaml' -exec rm -f {} + 2>/dev/null || true
