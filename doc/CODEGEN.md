# Code Generation

ECP generates both Go API types and Kubernetes CRD YAML from a single source of truth: the OpenAPI specification in the [go-sdk](https://github.com/eu-sovereign-cloud/go-sdk) submodule. This ensures the client library and the server share identical types with no encoding gap.

## Overview

Two separate generation pipelines run as part of `make generate-api`:

1. **Model generation** — copies Go types from the go-sdk OpenAPI schemas into `foundation/persistence/generated/types/`, applies Kubernetes annotations, and runs controller-gen to produce `DeepCopy` methods.
2. **CRD generation** — uses controller-gen to generate CRD YAML from the annotated Go types.

Generated files must never be edited by hand. CI enforces this with `make generate-api-verify`.

## OpenAPI-to-Go Type Generation

**Entry point:** `foundation/persistence/scripts/generate-model.sh`

**Steps:**
1. Reads `.go` schema files from `modules/go-sdk/pkg/spec/schema/`.
2. Copies each file to `foundation/persistence/generated/types/zz_generated_<filename>`.
3. Rewrites the package declaration to `package types`.
4. Injects `+kubebuilder:object:generate=true` and `+kubebuilder:object:root=true` annotations.
5. Replaces `time.Time` with `metav1.Time` and adjusts imports.
6. Normalizes map types (`map[string]interface{}` → `map[string]string`).
7. Fixes union fields without JSON tags for controller-gen compatibility.
8. Runs `gofmt` on the output.
9. Runs `controller-gen object` to generate `DeepCopy` methods alongside the types.

**Outputs:** `foundation/persistence/generated/types/`

## CRD Generation

**Entry point:** `make generate-crds` in `foundation/persistence/Makefile`

Runs `go generate -tags=crdgen ./...` inside `foundation/persistence/`. Source files tagged with `//go:build crdgen` invoke controller-gen to produce CRD YAML from Go struct annotations (`+kubebuilder:resource`, `+kubebuilder:validation`, etc.).

**Outputs:** `foundation/persistence/generated/crds/`

## Running Generation

```bash
# Generate everything (models + CRDs) — standard developer command
make generate-api

# Same, inside the tools container
make generate-api-ctzd

# CI gate — fails if the working tree is dirty after generation
# (used by the generate-api CI job; do not run this locally to iterate)
make generate-api-verify
```

`make generate-api` delegates to `make -C foundation/persistence generate-all`, which runs `generate-models` then `generate-crds` in sequence.

## Conventions

- Generated files are prefixed with `zz_generated` or placed under `generated/` directories.
- **Never edit generated files manually.** Changes will be overwritten on the next `make generate-api`.
- After changing OpenAPI specs in `modules/go-sdk`, run `make generate-api` and commit the result.
- CI runs `make generate-api-verify` in every PR; it fails if the committed generated files differ from a fresh run.

## Cleaning Generated Files

```bash
# Remove generated Go types
make -C foundation/persistence clean-generated

# Remove generated CRD YAML
make -C foundation/persistence clean-crds
```
