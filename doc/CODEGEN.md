# Code Generation

ECP generates both Go API types and Kubernetes CRD YAML from a single source of truth: the OpenAPI specification in the [go-sdk](https://github.com/eu-sovereign-cloud/go-sdk) submodule. This ensures the client library and the server share identical types with no encoding gap.

## Overview

ECP's type layer is built by two generation steps:

1. **Shared schema types** (`make generate-api`) — reads the go-sdk `resource.go` schema, emits `framework/backend/kubernetes/schema/v1/zz_generated_resource.go` (CRD-envelope types shared by all slices, aliased as `schemav1`), applies kubebuilder markers, and runs controller-gen to produce `DeepCopy` methods.
2. **Per-slice types** (`go generate ./...` in `resource/`) — each resource slice declares an explicit `//go:generate` directive in `backend/kubernetes/generate.go`; `model-gen` extracts the slice-specific types from the go-sdk schema and emits `zz_generated_schema.go` in that slice's `backend/kubernetes/` package.

**CRD YAML generation** is scaffolded (a `generate-crds` target in `framework/backend/kubernetes/Makefile`) but currently inactive — no `//go:build crdgen`-tagged sources exist yet.

Generated files must never be edited by hand. CI enforces this with `make generate-api-verify`.

## Generators

All code generators live at `framework/backend/kubernetes/cmd/`:

| Generator | Path | Purpose |
|-----------|------|---------|
| `model-gen` | `framework/backend/kubernetes/cmd/model-gen/` | Transforms go-sdk schema `.go` files into Kubernetes-compatible type definitions (`package kubernetes` for slices, `package v1` for shared schema types) |
| `conditioned-gen` | `framework/backend/kubernetes/cmd/conditioned-gen/` | Generates `zz_generated.conditions.go` for conditioned CR types |
| `inject-kubebuilder-markers` | `framework/backend/kubernetes/cmd/inject-kubebuilder-markers/` | Injects `+kubebuilder:*` annotations into type files |

Resource slices invoke `model-gen` via `//go:generate` directives in their `backend/kubernetes/generate.go`. Note: `make generate-api` orchestrates only the shared `framework/backend/kubernetes/schema/v1/` types — per-slice generation runs separately via `go generate ./...` in `resource/`.

## Shared Schema Types (`make generate-api`)

**Entry point:** `make generate-api` → `framework/backend/kubernetes generate-all`

`model-gen` runs in single-file mode against `modules/go-sdk/pkg/spec/schema/resource.go` — the go-sdk
schema that defines CRD-envelope types shared by all resource slices.

**Steps:**
1. `model-gen` reads `resource.go` and emits `framework/backend/kubernetes/schema/v1/zz_generated_resource.go` as `package v1`.
2. `inject-kubebuilder-markers` annotates the emitted types with `+kubebuilder:*` markers.
3. `controller-gen object` generates `zz_generated.deepcopy.go` alongside.

**Outputs:**
- `framework/backend/kubernetes/schema/v1/zz_generated_resource.go` — shared types (`CommonData`, `Conditioned`, `Reference`, `Zone`, `Cidr`, `IPVersion`, `VolumeReference`, etc.)
- `framework/backend/kubernetes/schema/v1/zz_generated.deepcopy.go`

All importers alias this package as **`schemav1`**.

## Per-Slice Types (`go generate ./...`)

Each resource slice has a `backend/kubernetes/generate.go` with an explicit `//go:generate` directive:

```
//go:generate go run .../framework/backend/kubernetes/cmd/model-gen \
  --schema-file=.../modules/go-sdk/pkg/spec/schema/<resource>.go \
  --output-file=zz_generated_schema.go \
  --package-name=kubernetes \
  --root-types=<Kind>Spec,<Kind>Status \
  --shared-types-source=.../modules/go-sdk/pkg/spec/schema/resource.go
```

`model-gen` extracts only the named root types (and their transitive dependencies) from the go-sdk schema.
Types present in `--shared-types-source` are qualified with the `schemav1` alias rather than re-emitted.

Run per-slice generation from the repo root:
```bash
(cd resource && go generate ./...)
```

**Steps per slice:**
1. `model-gen` reads the slice's go-sdk schema file and extracts the named `--root-types`.
2. Rewrites the package declaration to `package kubernetes`.
3. Replaces `time.Time` with `metav1.Time`, normalizes map types, and qualifies shared types with `schemav1`.
4. Injects `+kubebuilder:object:generate=true` and `+kubebuilder:object:root=true` annotations.
5. Runs `gofmt` on the output.
6. `controller-gen object` generates `zz_generated.deepcopy.go` alongside.

**Output per slice:**
- `resource/{group}/{resource}/vN/backend/kubernetes/zz_generated_schema.go`

## CRD Generation (planned)

**Entry point:** `make generate-api` → `framework/backend/kubernetes generate-crds` → `go generate -tags=crdgen ./...`

The scaffold exists — `framework/backend/kubernetes/Makefile` has a `generate-crds` target that would invoke
controller-gen to produce CRD YAML from Go struct `+kubebuilder:*` annotations. However, no
`//go:build crdgen`-tagged source files exist yet, so this step is currently a no-op.

**Planned output:** `framework/backend/kubernetes/crds/`

## Running Generation

```bash
# Generate shared schema types (framework/backend/kubernetes/schema/v1/)
make generate-api

# Generate per-slice types ( resource/.../.../backend/kubernetes/zz_generated_schema.go)
(cd resource && go generate ./...)

# Same, inside the tools container
make generate-api-ctzd

# CI gate — fails if the working tree is dirty after generation
# (used by the generate-api CI job; do not run this locally to iterate)
make generate-api-verify
```

## Adding Generated Types to a New Slice

When a go-sdk schema gains a new resource that needs a full slice:

1. Create the slice directory: `resource/<group>/<resource>/vN/`.
2. Add `domain.go` (`package v1`) with the canonical domain type and identity consts.
3. Add `backend/kubernetes/generate.go` with a `//go:generate` directive specifying `--root-types` for the new Kind and `--shared-types-source` pointing to go-sdk's `resource.go`.
4. Run `(cd resource && go generate ./...)` — `model-gen` emits `zz_generated_schema.go` in the new slice's `backend/kubernetes/`.
5. Add `frontend/rest/handler.go` and `frontend/rest/converter.go`.
6. Add `controller.go`, `plugin.go`, `plugin_handler.go` to `backend/kubernetes/`.

## Conventions

- Generated files are prefixed with `zz_generated`.
- **Never edit generated files manually.** Changes will be overwritten on the next generation run (`make generate-api` for shared `schema/v1/` types; `go generate ./...` in `resource/` for per-slice types).
- After changing OpenAPI specs in `modules/go-sdk`, run `make generate-api` (shared types) and `(cd resource && go generate ./...)` (per-slice types), then commit the result.
- CI runs `make generate-api-verify` in every PR; it fails if the committed generated files differ from a fresh run.

## Import Alias Convention

All generated and hand-written code follows the canonical `<resource><layer>` import-alias convention. Importas lint enforcement is **planned** (the `alias:` list in `.golangci.yml` is currently empty; aliases are followed by hand convention):

| Alias | Package |
|-------|---------|
| `bsdom` | `resource/storage/block-storage/v1` |
| `bsk8s` | `resource/storage/block-storage/v1/backend/kubernetes` |
| `bsrest` | `resource/storage/block-storage/v1/frontend/rest` |
| `netdom` | `resource/network/network/v1` |
| `netk8s` | `resource/network/network/v1/backend/kubernetes` |
| `wsdom` | `resource/workspace/v1` |
| `wsk8s` | `resource/workspace/v1/backend/kubernetes` |
| `rdom` | `resource/region/v1` |
| `rk8s` | `resource/region/v1/backend/kubernetes` |

The alias convention neutralizes deep paths at call sites — the full package path never appears raw in code. `model-gen` emits `schemav1` (the shared schema package alias) in generated import blocks automatically.
