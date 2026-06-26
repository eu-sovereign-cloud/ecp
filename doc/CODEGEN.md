# Code Generation

ECP generates both Go API types and Kubernetes CRD YAML from a single source of truth: the OpenAPI specification in the [go-sdk](https://github.com/eu-sovereign-cloud/go-sdk) submodule. This ensures the client library and the server share identical types with no encoding gap.

## Overview

ECP's type layer is built by two generation steps:

1. **Shared schema types** (`make generate-api`) — reads the go-sdk `resource.go` schema, emits `framework/backend/kubernetes/schema/v1/zz_generated_resource.go` (CRD-envelope types shared by all slices, aliased as `schemav1`), applies kubebuilder markers, and runs controller-gen to produce `DeepCopy` methods.
2. **Per-slice types** (`go generate ./...` in `resource/`) — each resource slice declares an explicit `//go:generate` directive in `backend/kubernetes/generate.go`; `model-gen` extracts the slice-specific types from the go-sdk schema and emits `zz_generated_schema.go` in that slice's `backend/kubernetes/` package.
3. **CRD YAML** (`make generate-api`) — injects `+kubebuilder:validation:*` markers into every slice's generated schema and runs controller-gen `crd` over all 18 resource slices, emitting CRD YAML files into `chart/crd/`.

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

**Outputs (generate-models):**
- `framework/backend/kubernetes/schema/v1/zz_generated_resource.go` — shared types (`CommonData`, `Conditioned`, `Reference`, `Zone`, `Cidr`, `IPVersion`, `VolumeReference`, etc.)
- `framework/backend/kubernetes/schema/v1/zz_generated.deepcopy.go`

**Outputs (generate-crds):**
- `chart/crd/*.yaml` — 18 CRD YAML files (one per resource slice; see [CRD Generation](#crd-generation))

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
- `resource/{group}/vN/{resource}/backend/kubernetes/zz_generated_schema.go`

## CRD Generation

**Entry point:** `make generate-api` → `framework/backend/kubernetes generate-crds`

`generate-crds` produces CRD YAML for all 18 resource slices in two steps:

1. **Inject kubebuilder markers** — `inject-kubebuilder-markers` is run over each slice's
   `resource/**/v1/**/backend/kubernetes/` directory. It reads the `x-kubebuilder-validation-*`,
   `x-kubebuilder-default`, and `x-cel-*` struct tags from `zz_generated_schema.go` and injects the
   corresponding `// +kubebuilder:validation:*` comment markers above each field. The tool is idempotent
   (strips prior markers before re-injecting), so CI's `generate-api-verify` gate always stays green.

2. **Run controller-gen** — `go tool sigs.k8s.io/controller-tools/cmd/controller-gen crd` is invoked with
   `paths="github.com/eu-sovereign-cloud/ecp/resource/..."` from the `framework/backend/kubernetes/`
   directory. The Go workspace (`go.work`) is active, so the cross-module shared types (`schemav1.CommonData`,
   etc.) resolve correctly. controller-gen v0.20.0 emits one YAML file per resource group+plural into
   `chart/crd/` (the repo-root CRD output directory).

**Output:** `chart/crd/*.yaml` — 18 flat CRD YAML files named `<group>_<plural>.yaml`:

```
chart/crd/
├── authorization.v1.secapi.cloud_role-assignments.yaml
├── authorization.v1.secapi.cloud_roles.yaml
├── compute.v1.secapi.cloud_instances.yaml
├── compute.v1.secapi.cloud_skus.yaml
├── network.v1.secapi.cloud_internet-gateways.yaml
├── network.v1.secapi.cloud_network-skus.yaml
├── network.v1.secapi.cloud_networks.yaml
├── network.v1.secapi.cloud_nics.yaml
├── network.v1.secapi.cloud_public-ips.yaml
├── network.v1.secapi.cloud_route-tables.yaml
├── network.v1.secapi.cloud_security-group-rules.yaml
├── network.v1.secapi.cloud_security-groups.yaml
├── network.v1.secapi.cloud_subnets.yaml
├── storage.v1.secapi.cloud_block-storages.yaml
├── storage.v1.secapi.cloud_images.yaml
├── storage.v1.secapi.cloud_skus.yaml
├── v1.secapi.cloud_regions.yaml
└── workspace.v1.secapi.cloud_workspaces.yaml
```

The CRDs carry full validation fidelity: CEL `x-kubernetes-validations`, `maxItems`/`maxLength`,
`enum`, and `default` constraints all survive from the go-sdk struct tags through the inject step
into the final YAML.

`generate-api-verify` (CI gate) runs `make generate-api` and asserts the working tree is clean across
`framework/backend/kubernetes/`, `resource/` (marker-laden schemas), and `chart/` (CRD YAML).

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

1. Create the slice directory: `resource/<group>/vN/<resource>/`.
2. Add `domain.go` (`package <resource>`) with the canonical domain type and identity consts.
3. Add `backend/kubernetes/generate.go` with a `//go:generate` directive specifying `--root-types` for the new Kind and `--shared-types-source` pointing to go-sdk's `resource.go`.
4. Run `(cd resource && go generate ./...)` — `model-gen` emits `zz_generated_schema.go` in the new slice's `backend/kubernetes/`.
5. Add `<resource>_converter.go` to the group's `resource/<group>/vN/frontend/rest/` directory; add handler methods to `<resource>_handler.go` (or create the group handler if this is the first resource in the group).
6. Add `controller.go`, `plugin.go`, `plugin_handler.go` to `backend/kubernetes/`.

## Conventions

- Generated files are prefixed with `zz_generated`.
- **Never edit generated files manually.** Changes will be overwritten on the next generation run (`make generate-api` for shared `schema/v1/` types, per-slice marker injection, and CRD YAML; `go generate ./...` in `resource/` for per-slice types).
- After changing OpenAPI specs in `modules/go-sdk`, run `make generate-api` (shared types + CRDs) and `(cd resource && go generate ./...)` (per-slice types), then commit the result.
- CI runs `make generate-api-verify` in every PR; it fails if the committed generated files differ from a fresh run.

## Import Alias Convention

All generated and hand-written code follows the canonical `<resource><layer>` import-alias convention. Importas lint enforcement is **planned** (the `alias:` list in `.golangci.yml` is currently empty; aliases are followed by hand convention):

| Alias | Package |
|-------|---------|
| `bsdom` | `resource/storage/v1/block-storage` |
| `bsk8s` | `resource/storage/v1/block-storage/backend/kubernetes` |
| `bsrest` | `resource/storage/v1/frontend/rest` |
| `netdom` | `resource/network/v1/network` |
| `netk8s` | `resource/network/v1/network/backend/kubernetes` |
| `wsdom` | `resource/workspace/v1` |
| `wsk8s` | `resource/workspace/v1/backend/kubernetes` |
| `rdom` | `resource/region/v1` |
| `rk8s` | `resource/region/v1/backend/kubernetes` |

The alias convention neutralizes deep paths at call sites — the full package path never appears raw in code. `model-gen` emits `schemav1` (the shared schema package alias) in generated import blocks automatically.
