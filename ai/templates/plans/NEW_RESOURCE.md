# Plan — Add a new vertical resource

This is the execution guide for adding (or completing) a **vertical resource** in ECP: a
self-contained slice that cuts through `domain → backend (CR + controller + plugin) →
frontend (REST)`, plus the dummy CSP plugin, deployment/RBAC, tests, and docs.

It is a **dependency-ordered checklist, not a script.** Do the steps in order; explore the
canonical references when a detail is unclear. Every resource is already *scaffolded* from
the go-sdk (see §3), but how much is scaffolded varies — so this guide lists **every** step
and §1 tells you how to skip the ones already done.

## Calling signature

```
/plan new ['read-only'] ('global'|'regional-tenant'|'regional-workspace') resource [<group>] <resource_name> from <resource_spec_path>
```

- `read-only` — present iff the resource exposes only `list`/`get` (no controller/plugin).
- scope — one of `global`, `regional-tenant`, `regional-workspace` (see §1.3).
- `<group>` — the API group; **present for regional resources, omitted for global** ones.
- `<resource_name>` — the resource's spec `name:` (see §1.1 for how it maps to dirs/Kind).
- `<resource_spec_path>` — a pointer to the group spec yaml; resolve it per §1.1 (the literal
  path may be approximate — the real files live under `modules/go-sdk/spec/spec/resources/`).

The declared `read-only` flag and scope are **claims to verify against the spec** (§1.3,
§1.4). If either disagrees with the spec, **stop and tell the user** before writing code.

---

## 0. Hard rules (read first)

1. **The SECA spec is the source of truth.** Resource name, fields, requirements, and
   validations come from the spec, vendored at
   [modules/go-sdk/spec/](../../../modules/go-sdk/spec/). The resource must strictly abide
   by it — including every validation (required/enum/min/max/CEL).
2. **Never edit the submodules** `modules/go-sdk/` or `modules/go-sdk/spec/`. CI fails if they
   are touched by hand. They are read-only inputs to code generation.
3. **Follow [doc/CONVENTIONS.md](../../../doc/CONVENTIONS.md)** for every hand-written line
   (summary in §6). Generated `zz_generated_*` files are exempt and never hand-edited.
4. **Static analysis alone will not fully explain code generation** (§3). When unsure, **run
   the generation step and observe which files change** rather than predicting it.
5. **If the requested resource cannot be found in the spec, stop and ask the user.** Do not
   invent a name, group, or schema.
6. **Verify only at the very end** (§7); the make output is long. **Docker and Kind are the
   only required tooling.** If they are unavailable, **warn the user and mark the work
   UNVERIFIED** — never claim checks passed that did not run.
7. **Commit policy is the user's call (§8).** Ask before committing; never auto-merge; never
   run anything that can erase existing work.

---

## 1. Preflight — resolve identity, validate the claims, compute the delta

Do all of this before writing code.

### 1.1 Resolve identity from the spec

Open the group spec. The canonical location is
`modules/go-sdk/spec/spec/resources/<group>.v1.yaml`; if `<resource_spec_path>` as given does
not exist, find the file there. The file's top-level `name:` is the **group**; under
`resources:` find the entry whose `name:` matches `<resource_name>`. From that entry derive,
**but confirm against the existing scaffold directory** (the scaffold already encodes the
correct strings — locate it rather than assuming):

| Artifact | How to derive | Examples |
|---|---|---|
| **Schema type** | the type at the end of the entry's `schema:` `$ref` | `Image`, `Subnet`, `BlockStorage`, `Region`, `StorageSku` |
| **Go Kind** | schema type, PascalCase, with initialisms (§6) applied | `Image`, `Subnet`, `BlockStorage`, `Region`, `StorageSKU`, `PublicIP`, `NIC` |
| **Slice dir** | kebab-case of the schema type | `image`, `subnet`, `block-storage`, `storage-sku` |
| **Plural** | the entry's `plural:`, spaces → hyphens | `images`, `subnets`, `block-storages`, `skus`, `public-ips` |
| **API group** | `<group>.v1.secapi.cloud` | `storage.v1.secapi.cloud`, `network.v1.secapi.cloud` |
| **CRD file** | `chart/crd/<apigroup>_<plural>.yaml` | `storage.v1.secapi.cloud_images.yaml` |

> ⚠️ **The spec `name:` is not always the Kind/dir.** Generic names are schema-qualified —
> e.g. spec `name: sku` in the storage group has schema `StorageSku`, so Kind is
> `StorageSKU`, dir is `storage-sku`, plural is `skus`. Always drive the Kind/dir from the
> **schema type**, and verify against the scaffold dir actually present under
> `resource/<group>/`.

**Slice path:** `resource/<group>/v1/<dir>/` (only `v1` exists — default to it).
**Exception — group equals resource:** when the group `name:` equals the resource `name:`
(e.g. `workspace`, `region`), the slice collapses to `resource/<resource>/v1/` with **no
group segment**. `region` has a further exception — see §5.

### 1.2 Confirm the generated Go schema exists

The slice's K8s types are generated from `modules/go-sdk/pkg/spec/schema/<schema>.go`, which
must already define `<Kind>Spec` (and `<Kind>Status` for read-write resources). If that file
is missing, the resource is not in the pinned go-sdk version. **Stop**: the submodule is
read-only — request a go-sdk bump; do not hand-author the schema.

### 1.3 Classify and validate the scope

Scope is fixed by the spec entry's `hierarchy:` field — it is **not a free choice**. Validate
the declared scope against it:

| Declared scope | spec `hierarchy` | domain embed | k8s scope | CR namespace | REST handler params | gateway |
|---|---|---|---|---|---|---|
| `regional-tenant` | `[tenant]` | `RegionalMetadata` | `Namespaced` | tenant only | `(tenant, …)` | regional |
| `regional-workspace` | `[tenant, workspace]` | `RegionalMetadata` | `Namespaced` | tenant+workspace | `(tenant, workspace, …)` | regional |
| `global` | `[]` | `Metadata` | `Cluster` | cluster-scoped | `(…)` no tenant/ws | global |

- **Deeper hierarchies** (e.g. `[tenant, workspace, network]`) still map to
  `regional-workspace`: the **k8s namespace caps at tenant+workspace**, and the extra parent
  (e.g. `network`) is modeled as a **Reference field in the domain/spec**, not a deeper
  namespace. Note the split: the **REST handler params follow the *full* hierarchy** — the
  go-sdk `ServerInterface` method carries the extra path param (e.g.
  `ListSubnets(…, tenant, workspace, network, params)`), which you use to set the parent
  reference — while the **namespace scope** stays tenant+workspace. There may be no
  fully-implemented exemplar for the extra level — derive it carefully and **tell the user the
  parent-reference handling is underspecified.**
- **If the declared scope disagrees with the hierarchy, notify the user** and ask how to
  proceed before continuing.

### 1.4 Validate read-only

Read the entry's `operations:`. `operations: [list, get]` ⇒ **read-only** (no write path, no
controller, no plugin). Anything including `put`/`delete` ⇒ **read-write**.

- **If the command's `read-only` flag disagrees with the spec operations, notify the user**
  and stop. (A `read-only` resource needs work only in the read path — see the per-step
  "read-only" notes; do not scaffold a controller/plugin for it.)

### 1.5 Identify the REST owner (per-API-group, not per-resource)

REST handlers are **per API group**. One slice in a group owns `frontend/rest/handler.go`,
which implements the *whole group's* `ServerInterface`; sibling slices contribute only a
`frontend/rest/converter.go`. Decide whether your resource is:
- a **new resource in an existing group** → add your converter and fill in your methods on
  the group owner's handler; or
- a **new group** → create a new handler implementing that group's `ServerInterface` and
  register a new gateway block (§4.11).

### 1.6 Compute the delta

List the slice dir and compare against a complete vertical of the same shape (§ References).
Do only the **missing** steps. A slice can be a bare scaffold (only
`backend/kubernetes/{generate.go, resource.go, zz_generated_*}`), partially built, or
complete. A code-complete slice may still be **incompletely wired** downstream (gateway, dummy
`main.go`, RBAC across *all* ClusterRoles, tests) — verify those separately. When unsure
whether a generated file is current, regenerate (§3) and observe; don't trust a stale tree.

### References — canonical verticals (by role)

| Role | Use as the exemplar for | Slice |
|---|---|---|
| **regional-tenant, read-write** | full stack incl. controller/plugin | `resource/workspace/v1/` |
| **regional-workspace, read-write** | full stack incl. controller/plugin | `resource/storage/v1/block-storage/` |
| **global, read-only** | global scope + the read-only (rest-only) shape | `resource/region/v1/` |

Use **workspace** (`resource/workspace/v1/`) and **block-storage** (`resource/storage/v1/block-storage/`) as the controller/plugin references. Use **region** (`resource/region/v1/`)
only for the global-scope shape and the read-only shape — it has no controller/plugin. If a
read-write **global** resource is ever requested, take scope from `region` but controller/
plugin from the regional exemplars.

---

## 2. Repository map (so you don't have to re-explore)

A few sentences per directory you will touch:

- **`resource/<group>/v1/<dir>/`** — the vertical slice. `domain.go` holds the canonical
  domain type and identity constants (`package <dir-as-identifier>`). `backend/kubernetes/` holds the CR wrapper, conversion,
  controller, plugin interface + handler, and generated files. `resource/<group>/v1/frontend/rest/` holds the
  REST↔domain converters (`<resource>_converter.go`) and (for the group owner) the HTTP handlers (`<resource>_handler.go`) — one handler per API group shared across all resources in that group.
- **`resource/common/`** — shared domain types (`Reference`, `Status`, `ResourceState`,
  `RegionalMetadata`/`Metadata`) and shared conversion helpers (`commonbackend`,
  `commondomain`). Reuse these; never re-implement state conversion or metadata.
- **`framework/`** — the resource-agnostic SDK. `framework/backend/kubernetes/` has the repo/
  reader/writer adapters, the generic controller and plugin handler, the codegen tools under
  `cmd/`, and **the Makefile with the hardcoded slice list** (§3). `framework ↛ resource` is
  compiler-enforced — framework never names a concrete resource.
- **`gateway/cmd/`** — the API-server binaries. `regionalapiserver.go` wires the
  tenant/workspace-scoped group handlers; `globalapiserver.go` wires the global handler.
- **`chart/crd/`** — generated CRD YAML, one file per group+plural. Output only.
- **`csp/dummy/`** — the reference CSP plugin (mock backend, no real cloud). `pkg/plugin/` has
  one file per resource that has a controller; `cmd/main.go` wires them; `deploy/` holds
  manifests + RBAC; `test/integration/` holds plugin integration tests (build-tagged, Kind).
  **Only extend `dummy`** — `ionos`/`aruba` need custom implementations against their own
  operators; leave them alone.
- **`test/e2e/`** — full-stack harness, split by component:
  `test/e2e/test/integration/{delegator,gateway-regional,gateway-global}/` and
  `test/e2e/deploy/{delegator,gateway-regional,gateway-global,...}/` manifests.
- **`doc/`** — `CONVENTIONS.md` (mandatory coding standards), `CODEGEN.md`, `PLUGINS.md`,
  `ARCHITECTURE.md`, `CI_DEVEX.md`.

---

## 3. Code generation — how spec becomes types + CRDs (and the silent trap)

Pipeline:
`spec yaml (upstream) → modules/go-sdk/pkg/spec/schema/<schema>.go → (model-gen, per slice)
zz_generated_schema.go → (inject-kubebuilder-markers) markers → (controller-gen crd)
chart/crd/<apigroup>_<plural>.yaml`. Two entry points, **both needed** for a slice:

- `(cd resource && go generate ./...)` — runs each slice's `//go:generate` directives
  (`model-gen` → `zz_generated_schema.go`; `mockgen` → the test mocks). Driven by
  `backend/kubernetes/generate.go`.
- `make generate-api` — shared schema types **and** CRD generation (marker injection over the
  slice list, then `controller-gen crd`).

> ### ⚠️ The hardcoded slice list (the one Makefile edit you may need)
> `controller-gen crd` auto-discovers your CRD via the glob `paths="…/resource/..."`, **but
> kubebuilder-marker injection runs off a hardcoded list** in
> [framework/backend/kubernetes/Makefile](../../../framework/backend/kubernetes/Makefile)
> (the `for dir in … done` loop in `generate-crds`). Each entry has the form
> `"$(REPO_ROOT)/resource/<group>/v1/<dir>/backend/kubernetes"`. If your slice's `backend/kubernetes` dir
> is **not** in that loop, the CRD still ships — but **silently without its spec validations**
> — and `generate-api-verify` stays green because the tree is self-consistent. This is the
> single most common silent failure. **Check whether your slice is already in the list; add it
> only if missing.** This is the *only* Makefile edit a vertical needs.

> ### Don't trace generation statically
> Some steps have no visible invocation (e.g. `conditioned-gen`, which emits
> `zz_generated.conditions.go` for `+ecp:conditioned` CR types, and marker→CRD lowering by
> controller-gen). After generating, **inspect the outputs**: confirm `zz_generated_schema.go`,
> `zz_generated.deepcopy.go`, `zz_generated.conditions.go` (read-write only), and
> `chart/crd/<apigroup>_<plural>.yaml` exist and that the CRD carries your spec's validations.

Reference: [doc/CODEGEN.md](../../../doc/CODEGEN.md).

---

## 4. Build steps (dependency order)

> **Types → CRD → conversion → controller/plugin → REST → wiring → tests → docs.**

Notation: `<Kind>` = Go Kind, `<dir>` = slice dir, `<plural>` = spec plural. Skip the marked
steps for **read-only** resources.

### 4.1 Domain type — `resource/<group>/v1/<dir>/domain.go`
Identity constants (`Kind`, `Resource`=`<plural>`, `Group`, `Version`, `ProviderID`) and the
canonical domain struct. Package name is `package <dir-as-identifier>` (e.g. `package blockstorage`,
`package storagesku`, `package network`). Embed `RegionalMetadata` (regional) or `Metadata` (global).
Define `<Kind>Spec`; define `<Kind>Status` (embeds `domain.Status`) **only for read-write**
resources — read-only resources have no Status. Ref: the matching canonical vertical's
`domain.go`.

### 4.2 CR wrapper — `resource/<group>/v1/<dir>/backend/kubernetes/resource.go`
**This is where the spec's requirements/validations land**, as kubebuilder markers (mostly
auto-injected from go-sdk struct tags). Define group/version constants, `<Kind>GVR`/`<Kind>GVK`,
the `SchemeBuilder`/`AddToScheme`, the CR struct (`TypeMeta`+`ObjectMeta`+`Spec`+
`schemav1.CommonData`+`*Status`), the `<Kind>List`, and `init()` registration. Set the
kubebuilder markers, **including the correct `scope=`** (`Namespaced` regional / `Cluster`
global) and `+ecp:conditioned` **iff read-write**. A scaffold usually already has this — verify
it, don't recreate it. Ref: the matching vertical's `resource.go`.

### 4.3 Generate directives — `resource/<group>/v1/<dir>/backend/kubernetes/generate.go`
`//go:generate` for `model-gen` (`--root-types=<Kind>Spec,<Kind>Status` for read-write, just
`<Kind>Spec` for read-only) and — **read-write only** — `mockgen` for `Repo` and the
`<Kind>Plugin`. Ref: block-storage (read-write) / storage-sku (read-only) `generate.go`.

### 4.4 Register in the Makefile slice list — `framework/backend/kubernetes/Makefile`
Per the §3 trap: add `"$(REPO_ROOT)/resource/<group>/v1/<dir>/backend/kubernetes" \` to the
`generate-crds` loop **only if it is not already there.**

### 4.5 Generate and inspect
Run `(cd resource && go generate ./...)` and `make generate-api`. Confirm the generated files
and the CRD validations per §3. Do not proceed until the CRD reflects the spec.

### 4.6 Conversion — `resource/<group>/v1/<dir>/backend/kubernetes/conversion.go`
`<Kind>FromCR(obj client.Object) (*<dom>, error)` and `<Kind>ToCR(x *<dom>) (client.Object,
error)` (read-only resources still need `FromCR` for the reader adapter; keep `ToCR` for
symmetry/tests as the read-only exemplars do). Use `commonbackend.ResourceStateFromCR` (never a
raw cast) and the error template `"<resource> <name>: <description>: %w"`. **The CR namespace is
set here and encodes the scope** (§5). Ref: `resource/workspace/v1/backend/kubernetes/conversion.go` /
`resource/storage/v1/block-storage/backend/kubernetes/conversion.go` /
`resource/storage/v1/storage-sku/backend/kubernetes/conversion.go`.

### 4.7 Plugin interface — `resource/<group>/v1/<dir>/backend/kubernetes/plugin.go` *(read-write)*
`type <Kind>Plugin interface { … }`. Methods follow the spec operations: `Create` + `Delete` at
minimum, plus any resource-specific mutating verb. **Skip for read-only.** Ref: block-storage
(extra verb) / workspace (create+delete only) `plugin.go`.

### 4.8 Plugin handler — `…/plugin_handler.go` *(read-write)*
The reconciliation state machine. Embed the framework `GenericPluginHandler`; hold `repo` +
`plugin`; implement `HandleReconcile` and the `isX*`/`wantX*` predicates. **Structural symmetry
is mandatory** (CONVENTIONS §8): `isXPending` treats nil status as pending and only marks
deleting when `DeletedAt == nil`. Use **workspace** (`resource/workspace/v1/backend/kubernetes/`) as
the template for create/delete-only resources, **block-storage**
(`resource/storage/v1/block-storage/backend/kubernetes/`) when there is an extra mutating
operation. **Skip for read-only.**

### 4.9 Controller — `resource/<group>/v1/<dir>/backend/kubernetes/controller.go` *(read-write)*
`NewController(ctrlClient, dynClient, plugin, opts...)` wiring a repo adapter, the plugin
handler, and the framework generic controller. **Skip for read-only.** Ref: the matching
vertical's `controller.go`.

### 4.10 REST — `resource/<group>/v1/frontend/rest/`
REST is **per API group**, not per resource. The group's `frontend/rest/` directory is shared
by all resources in that group.
- Always add `<dir>_converter.go`: `<Kind>ToAPI`/`<Kind>ToAPIWithVerb`, `<Kind>IteratorToAPI`, a
  list-param helper, and (read-write only) `<Kind>FromAPI`. Read-only converters expose list/get
  shapes only (no `FromAPI`).
- **New resource in an existing group:** implement its `List/Get` (+ `CreateOrUpdate/Delete`
  for read-write) methods in `<dir>_handler.go` on the **group's** shared handler struct, and add
  a reader (and, for read-write, writer) field to that handler struct. **Match the go-sdk
  `ServerInterface` method signatures exactly** — they reflect the full spec path depth, not just
  the namespace scope: regional-tenant `(tenant, …)`, regional-workspace `(tenant, workspace, …)`,
  and a deeper resource carries its extra parent param (e.g. subnet's
  `(tenant, workspace, network, …)`).
- **New group:** add `handler.go` defining the group `Handler` struct and implementing the group's
  `ServerInterface`; per-resource methods go in their `<dir>_handler.go` file.

### 4.11 Gateway wiring — `gateway/cmd/regionalapiserver.go` (regional) / `globalapiserver.go` (global)
Build `k8sadapter.NewReaderAdapter` (always) and `NewWriterAdapter` (read-write only) for the
resource (GVR + `FromCR`/`ToCR`), and either add them to the existing group handler struct
(`resource/<group>/v1/frontend/rest/`), or register a new `…api.HandlerWithOptions(&<group>rest.Handler{…},
…BaseURL: "/providers/seca.<group>")` block for a new group. Ref: the storage block in
`gateway/cmd/regionalapiserver.go`; the region block in `gateway/cmd/globalapiserver.go`.

### 4.12 Dummy plugin — `csp/dummy/pkg/plugin/<dir>.go` + `csp/dummy/cmd/main.go` *(read-write)*
- New file `csp/dummy/pkg/plugin/<dir>.go`: a `type <Kind> struct{ logger }`, `New<Kind>`, and
  the `<Kind>Plugin` methods, each delegating to a `simulate<Kind>` helper added to
  `csp/dummy/pkg/plugin/simulate.go` (mirror `simulateBS`: stamp an expiry annotation, return
  `ErrStillProcessing` until it elapses).
- In `csp/dummy/cmd/main.go`: add the `<k8s>` import (pointing to
  `resource/<group>/v1/<dir>/backend/kubernetes`), `utilruntime.Must(<k8s>.AddToScheme(scheme))`
  in `init()`, instantiate the plugin, and `controllerSet.Add(<k8s>.NewController(…))`.
- **Skip entirely for read-only** (no controller to run).

### 4.13 Deployment & RBAC (permissions)
CRDs install automatically from `chart/crd/`. Add API-group rules to **every** relevant
ClusterRole — these drift, so add your resource wherever its peers appear and verify each role:
- **read-write:** two rules (`<plural>` and `<plural>/status`) on the **dummy delegator**
  ([csp/dummy/deploy/clusterrole.yaml](../../../csp/dummy/deploy/clusterrole.yaml)) and the
  **e2e delegator** ([test/e2e/deploy/delegator/clusterrole.yaml](../../../test/e2e/deploy/delegator/clusterrole.yaml))
  with full verbs; on the **e2e gateway-regional**
  ([test/e2e/deploy/gateway-regional/clusterrole.yaml](../../../test/e2e/deploy/gateway-regional/clusterrole.yaml))
  split read/write (`<plural>` full verbs, `<plural>/status` read-only).
- **read-only, regional:** read-only verbs on `<plural>` (no `/status`) in the e2e delegator and
  e2e gateway-regional roles; **no dummy delegator rule** (no controller).
- **read-only, global:** read-only verbs on `<plural>` in the e2e gateway-global role
  ([test/e2e/deploy/gateway-global/clusterrole.yaml](../../../test/e2e/deploy/gateway-global/clusterrole.yaml)).

### 4.14 Tests
Follow existing conventions; **examples are inline fixtures inside the tests — there is no
separate examples folder.** Add scenarios you judge worth covering for this resource (e.g. an
extra mutating verb deserves its own test).
- **Unit** (per slice, `resource/<group>/v1/<dir>/backend/kubernetes/*_test.go`, read-write):
  `conversion_test.go`, `plugin_handler_test.go`, `controller_test.go`, and an envtest
  (`<dir>_envtest_test.go` + `setup_envtest_test.go`). The plugin/repo mocks come from the
  `generate.go` mockgen lines. Read-only slices test conversion/REST only.
- **Dummy integration** (`csp/dummy/test/integration/<dir>_test.go`, read-write): build tag
  `//go:build integration`; create via the repo adapter; poll with
  `wait.PollUntilContextTimeout` for the expected `ResourceState`. Wire the repo + scheme in
  `main_test.go`. Ref: `csp/dummy/test/integration/blockstorage_test.go` + `main_test.go`.
- **E2E** (`test/e2e/test/integration/…`): `delegator/` (controller behavior, read-write),
  `gateway-regional/` (regional REST), `gateway-global/` (global REST). Read-only resources get
  a gateway read test only (mirror `test/e2e/test/integration/gateway-regional/storage_sku_test.go` /
  `gateway-global/regions_test.go`).

### 4.15 Documentation (avoid doc rot — this is critical)
- [README.md](../../../README.md) — update the layout/CRD-count if your change affects it (directory structure uses `<group>/vN/<resource>/`).
- Per-folder READMEs you touched — e.g. [csp/dummy/README.md](../../../csp/dummy/README.md),
  [test/e2e/README.md](../../../test/e2e/README.md) — update any resource list/behavior prose
  that mentions the resources by name.
- [doc/CODEGEN.md](../../../doc/CODEGEN.md) / [doc/PLUGINS.md](../../../doc/PLUGINS.md) — update
  only if you changed the generation pipeline or plugin contract (normally you don't).

---

## 5. Scope, group, region — quick reminders and scope-inference pointers

- **Scope is the spec `hierarchy`** (§1.3), not a free choice. **REST is per group** (§1.5).
- **Region is special:** slice at `resource/region/v1/` (no group dir), bare group
  `v1.secapi.cloud`, `scope=Cluster`, **no plugin/controller**, served by the **global**
  gateway. Most resources follow `resource/<group>/v1/<dir>/` with group `<group>.v1.secapi.cloud`.

**Scope-inference pointers — starting points; verify before relying** (these reflect a moment
in time and can drift):
- The differentiator between regional-tenant and regional-workspace is **namespace placement,
  not the metadata type** — both embed `RegionalMetadata`. Regional-tenant CRs live in the
  **tenant** namespace via a `tenantOnlyScope` helper in the slice's `conversion.go` (see
  workspace); regional-workspace CRs use the full **tenant+workspace** scope (see block-storage,
  `ComputeNamespace(<x>)`).
- The REST handler signature also reveals scope: a regional-tenant `List…(tenant, …)` takes no
  workspace param, while a regional-workspace `List…(tenant, workspace, …)` does.

---

## 6. Conventions (doc/CONVENTIONS.md — non-negotiable)

- **Conversion naming:** `XFromCR`/`XToCR`, `XFromAPI`/`XToAPI`, `XIteratorToAPI`,
  `XToAPIWithVerb` — never `Map`/`Domain`/`CR` as infix tokens.
- **Initialisms** always fully capitalised in hand-written names:
  `API CR CIDR GB HTTP ID IOPS IP IPv4 IPv6 SKU URL` (`StorageSKU`, `PublicIP`, `NIC`, not
  `StorageSku`/`PublicIp`/`Nic`). Generated `…Spec`/`…Status` lowercase tails are an accepted
  residual — do not "fix" them.
- **Typed-short variables/receivers** (`bs`, `ws`, `n`, `r`, `sku`); never shadow an import
  alias (`domain`, `resource`). Consistent receiver per type.
- **Structural symmetry:** parallel operations share helpers, names, and the error template
  `"<resource> <name>: <description>: %w"`. Use `commonbackend.ResourceStateFromCR`, not raw
  casts. Match the `isXPending` predicate across handlers.
- **Doc comment** on every exported symbol, beginning with its name; no package-name stutter
  (`const Kind`, not `const BlockStorageKind`, inside the slice package).
- **Import-alias convention:** `<resource><layer>` (`bsdom`/`bsk8s`/`bsrest` where `bsrest` points to
  `resource/storage/v1/frontend/rest`); `schemav1` for the shared schema package.

---

## 7. Verify (last step only — output is long, run once at the end)

```bash
make pre-commit-ctzd     # generate-api-verify + test + lint + gofmt + vuln + gosec, in Docker
make pre-merge-ctzd      # the above + rebase/workspace verification
```

Then run the **dummy plugin tests** and confirm they behave as expected (these need **Kind**):

```bash
make -C csp/dummy test-integration   # spins up a KIND cluster, runs the //go:build integration tests
```

`-ctzd` runs the target inside the tools container, so **Docker** covers the make checks and
**Kind** covers the dummy integration tests; no other toolchain is required and no extra
Makefile wiring is needed beyond the slice list (§4.4). **If Docker or Kind is unavailable,
stop, warn the user, and mark the resource UNVERIFIED** — state plainly that the checks did not
run.

---

## 8. Commit policy (ask the user)

Before doing anything with git, **verify the current branch is a good fit** for this work and
**check for uncommitted changes**. Ask the user whether they want a new branch and what to do
with any existing work. **Never run anything that can erase work** (no `git push --force`, no
discarding uncommitted changes without consent). **Never merge automatically.**

Then ask which commit policy they want:
1. **Auto-commit everything** — you categorize the files into commits as you see fit.
2. **Don't auto-commit** — leave everything staged/unstaged for the user.
3. **Ask exactly** — confirm each commit's contents and message with the user.

All commit messages use **Conventional Commits**, a **single line**, no co-author/agent
attribution (e.g. `feat(storage/image): implement image vertical`).

---

## Definition of done

- [ ] Identity resolved from the spec; `read-only` flag and scope validated against the spec
      (user notified on any mismatch); only-missing steps identified.
- [ ] `domain.go`, `resource.go`, `generate.go` present and correct (Status/`+ecp:conditioned`
      only for read-write).
- [ ] Slice present in the `framework/backend/kubernetes/Makefile` `generate-crds` loop (path form: `$(REPO_ROOT)/resource/<group>/v1/<dir>/backend/kubernetes`).
- [ ] Generation run; `zz_generated_*` and `chart/crd/<apigroup>_<plural>.yaml` present **with
      the spec's validations**.
- [ ] `conversion.go` present; `plugin.go`/`plugin_handler.go`/`controller.go` present for
      read-write (skipped for read-only).
- [ ] REST converter (+ handler methods on the group owner or a new handler) implemented.
- [ ] Gateway wired (regional or global; reader always, writer for read-write).
- [ ] Dummy plugin file + `main.go` registration done (read-write only).
- [ ] RBAC rules added to all relevant ClusterRoles per §4.13.
- [ ] Unit + integration (+ e2e) tests added, following existing conventions.
- [ ] Touched-folder READMEs / docs updated.
- [ ] `make pre-commit-ctzd`, `make pre-merge-ctzd`, and the dummy integration tests pass — or
      the work is marked UNVERIFIED with the reason.
- [ ] Commit policy followed per the user's choice; no auto-merge; no destructive git.
