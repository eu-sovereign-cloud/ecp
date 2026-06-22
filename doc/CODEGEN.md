# Code Generation

ECP generates both Go API types and Kubernetes CRD YAML from a single source of truth: the OpenAPI specification in the [go-sdk](https://github.com/eu-sovereign-cloud/go-sdk) submodule. This ensures the client library and the server share identical types with no encoding gap.

## Overview

Two separate generation pipelines run as part of `make generate-api`:

1. **Model generation** — reads Go types from the go-sdk OpenAPI schemas, routes each type to its destination, applies Kubernetes annotations, and runs controller-gen to produce `DeepCopy` methods.
2. **CRD generation** — uses controller-gen to generate CRD YAML from the annotated Go types.

Generated files must never be edited by hand. CI enforces this with `make generate-api-verify`.

## Generators

All code generators live at `framework/persistence/cmd/`:

| Generator | Path | Purpose |
|-----------|------|---------|
| `model-gen` | `framework/persistence/cmd/model-gen/` | Routes OpenAPI types to slice `backend/kubernetes/` or `framework/persistence/kubernetes/schema/v1/` |
| `conditioned-gen` | `framework/persistence/cmd/conditioned-gen/` | Generates `zz_generated.conditions.go` for conditioned CR types |
| `inject-kubebuilder-markers` | `framework/persistence/cmd/inject-kubebuilder-markers/` | Injects `+kubebuilder:*` annotations into type files |

Resource slices invoke the generators via `//go:generate` directives in their `backend/kubernetes/` package. The root `make generate-api` orchestrates the full pipeline.

## OpenAPI-to-Go Type Generation

**Entry point:** `make generate-api` → `go generate ./...` in `resources/`

**Routing logic:**

`model-gen` applies per-declaration routing:
- Each `<Kind>Spec` / `<Kind>Status` type (and version-local subtypes) → the matching resource slice at `resources/{scope}/{group}/{resource}/vN/backend/kubernetes/zz_generated_model.go`, as `package kubernetes`.
- Shared types used by 2+ resources (`StatusCondition`, `Reference`, `ResourceState`, `CommonData`, `Conditioned`, `Cidr`, `IPVersion`, `Zone`, `VolumeReference`, `*Metadata` family) → `framework/persistence/kubernetes/schema/v1/`.
- Unmatched Kinds and helper files → `resources/staging/` (holding area; not for production import).

**Steps per file:**
1. Reads `.go` schema files from `modules/go-sdk/pkg/spec/schema/`.
2. Routes each type declaration to its destination package.
3. Rewrites the package declaration to match the destination (`package kubernetes` or `package v1`).
4. Injects `+kubebuilder:object:generate=true` and `+kubebuilder:object:root=true` annotations.
5. Replaces `time.Time` with `metav1.Time` and adjusts imports.
6. Normalizes map types and fixes union fields for controller-gen compatibility.
7. Emits the canonical import alias for the destination package in generated import blocks.
8. Runs `gofmt` on the output.
9. Runs `controller-gen object` to generate `DeepCopy` methods alongside the types.

**Outputs:**
- `resources/{scope}/{group}/{resource}/vN/backend/kubernetes/zz_generated_model.go`
- `framework/persistence/kubernetes/schema/v1/` (shared types)
- `resources/staging/` (unmatched types)

## CRD Generation

**Entry point:** `make generate-api` → `go generate -tags=crdgen ./...` inside `resources/`

Source files tagged with `//go:build crdgen` invoke controller-gen to produce CRD YAML from Go struct annotations (`+kubebuilder:resource`, `+kubebuilder:validation`, etc.).

**Outputs:** `framework/persistence/kubernetes/crds/vN/`

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

## Promoting a Staging Resource

When the go-sdk adds a new resource Kind that has no wrapper slice yet, `model-gen` emits it to `resources/staging/<group>/<resource>/vN/backend/kubernetes/`. To promote it to a full slice:

1. Move the staging directory to `resources/{global,regional}/<group>/<resource>/vN/backend/kubernetes/`.
2. Add `domain/domain.go` with the canonical domain type and identity consts.
3. Add `frontend/rest/handler.go` and `frontend/rest/converter.go`.
4. Add `controller.go`, `plugin.go`, `plugin_handler.go` to `backend/kubernetes/`.
5. Re-run `make generate-api` — `model-gen` will route directly to the new slice.

## Conventions

- Generated files are prefixed with `zz_generated`.
- **Never edit generated files manually.** Changes will be overwritten on the next `make generate-api`.
- After changing OpenAPI specs in `modules/go-sdk`, run `make generate-api` and commit the result.
- CI runs `make generate-api-verify` in every PR; it fails if the committed generated files differ from a fresh run.

## Import Alias Convention

All generated and hand-written code uses the canonical `<resource><layer>` import-alias convention, enforced by `golangci-lint importas`:

| Alias | Package |
|-------|---------|
| `bsdom` | `resources/regional/storage/block-storages/v1/domain` |
| `bsk8s` | `resources/regional/storage/block-storages/v1/backend/kubernetes` |
| `bsrest` | `resources/regional/storage/block-storages/v1/frontend/rest` |
| `netdom` | `resources/regional/network/networks/v1/domain` |
| `netk8s` | `resources/regional/network/networks/v1/backend/kubernetes` |
| `wsdom` | `resources/regional/workspace/v1/domain` |
| `wsk8s` | `resources/regional/workspace/v1/backend/kubernetes` |
| `rdom` | `resources/global/regions/v1/domain` |
| `rk8s` | `resources/global/regions/v1/backend/kubernetes` |

The alias convention neutralizes deep paths at call sites — the full package path never appears raw in code. Aliases are declared in `.golangci.yml` and `model-gen` emits them automatically in generated import blocks.
