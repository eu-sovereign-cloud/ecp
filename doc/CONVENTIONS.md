# Go Style Conventions

This guide defines the authoritative naming and structural conventions for hand-written Go code in
`resource/`, `csp/`, `gateway/`, and `test/`. Generated files (`zz_generated_*`) are exempt — they
follow upstream tooling (go-sdk, controller-gen, conditioned-gen) and are never edited by hand.

Where a generated or SDK name disagrees with this guide, hand-written ecp code follows this guide.
See the [Known residuals](#appendix--known-residuals) appendix for accepted mismatches.

This guide extends (and does not replace) the import-alias convention in
[CONTRIBUTING.md § Import Alias Convention](CONTRIBUTING.md#import-alias-convention) and the linting
configuration in `.golangci.yml`.

---

## §1 — Package-name stutter

A type name must not repeat its package name. The package qualifier already disambiguates at call sites.

```go
// ✓ correct
domain.Reference
domain.Status
domain.StatusCondition
domain.ResourceState

// ✗ wrong — the word "Domain" is the package, so it is redundant
domain.ReferenceDomain
domain.StatusDomain
domain.StatusConditionDomain
domain.ResourceStateDomain
```

If two packages export a type with the same short name, the import alias (§ Import-alias convention)
provides disambiguation — do not inflate the type name to carry package information.

---

## §2 — Conversion-function naming

There is exactly one naming scheme for functions that convert between a domain type and another
representation. Use symmetric `XFromY` / `XToY` patterns; never use `Map`, `Domain`, or `CR` as
infix tokens in function names.

### Backend (Kubernetes CR ↔ domain)

| Direction | Signature shape |
|-----------|----------------|
| CR → domain | `XFromCR(obj client.Object) (*dom.X, error)` |
| domain → CR | `XToCR(x *dom.X) (client.Object, error)` |

### Frontend (REST API ↔ domain)

| Direction | Signature shape |
|-----------|----------------|
| API → domain | `XFromAPI(sdk …, id, region string) (*dom.X, error)` |
| domain → API | `XToAPI(x *dom.X) *sdk.X` |
| domain list → API list | `XIteratorToAPI(iter …) []sdk.X` |
| domain → API with HTTP verb | `XToAPIWithVerb(x *dom.X, verb string) *sdk.X` |

**Only the inbound direction returns an error**, and the asymmetry is deliberate. `XFromAPI` is
the request boundary: its input came off the wire and may name something the domain has no
representation for, so it has to be able to say so — `framework/frontend/rest.APIToDomain` is
typed for it and `HandleUpsert` turns the failure into an RFC 7807 response. `XToAPI` converts a
value that already passed a boundary check (`XFromAPI` on the write path, `XFromCR` on the read
path), so an error return there would be a branch no input can reach. A converter whose body has
nothing to reject still carries the error in its signature — the contract belongs to the layer,
not to whichever resource currently happens to have an enum field.

### Sub-object helpers in `resource/common`

The same `FromCR`/`ToCR`/`FromAPI`/`ToAPI` suffixes apply to sub-object converters:

```
ReferenceFromCR / ReferenceToCR / ReferenceFromAPI / ReferenceToAPI / ReferencePtrToAPI
StatusConditionFromCR / StatusConditionToCR / ConditionsFromCR / ConditionsToCR
ResourceStateFromCR / ResourceStateToCR / ResourceStateToAPI
IPVersionFromCR / IPVersionToCR / IPVersionFromAPI / IPVersionToAPI
ConditionsToAPI / conditionToAPI   (conditionToAPI is unexported — lower-case is intentional)
```

The enum and condition helpers follow §10: the ones that read a stored or requested value
(`…FromCR`, `…FromAPI`) return `(T, error)` and reject a value they do not recognise; the ones
that write it out (`…ToAPI`) are total. `ResourceStateToCR` is the exception on the write side —
the CRD requires the field, so there is no empty state to write and it reports one.

### The exported pair — `Converter`

Every slice's `backend/kubernetes/conversion.go` exports its two directions bundled as one value:

```go
// Converter is the CR<->domain conversion pair for BlockStorage, so a call site names one value
// instead of pairing the two directions by hand.
var Converter = k8sadapter.TwoWayConverter[*bsdom.BlockStorage]{
	FromCR: BlockStorageFromCR,
	ToCR:   BlockStorageToCR,
}
```

`TwoWayConverter[T]` lives in `framework/backend/kubernetes` — the only layer where
`client.Object` is legal — and every adapter that needs both directions
(`NewWriterAdapter`, `NewRepoAdapter`, and the two namespace-managing variants) takes it in place
of two function arguments. Read-only adapters (`NewReaderAdapter`, `NewWatcherAdapter`) keep the
bare `K8sToDomain[T]`: requiring a `ToCR` they never call would be an argument that exists only
to be ignored.

The win is at the call sites, not in the type. `gateway/cmd/regionalapiserver.go`,
`csp/dummy/pkg/plugin/`, each slice's `controller.go` and every suite that builds a repo used to
name `XToCR` and `XFromCR` separately at each of them; now they name `Converter` once, and a slice
that changes how it converts changes one line.

### Rationale

`MapCRToBlockStorageDomain` requires parsing three fused concepts at once. `BlockStorageFromCR` is
self-evident: `BlockStorage` is the result type, `FromCR` is the direction. The symmetric pair
`BlockStorageToCR` follows by inspection. Renaming all 6 slices to this scheme removes 4 legacy
templates and makes the direction unambiguous at every call site.

---

## §3 — Internal-identifier consistency

The same concept must have the same name everywhere it appears. Prefer the shortest name that
unambiguously identifies the concept in its scope.

### Canonical names

| Concept | Canonical name | Avoid |
|---------|---------------|-------|
| Storage capacity | `sizeGB` | `size`, `diskSize`, `volumeSize` |
| Domain reference field | `ref` | `reference`, `domRef` |
| Resource state | `state` | `resourceState`, `crState`, `domState` |
| Single condition in a loop | `c` | `cond`, `condition`, `domainStatusCondition` |
| Condition slice | `conds` | `conditions`, `crConditions` |
| Kubernetes resource version | `resourceVersion` | `resVersion`, `rv` |
| Domain object pointer | typed-short (see below) | `domain`, `dom`, `res` |
| REST path segment | `resourcePath` | `resource` (shadows import) |

### Typed-short domain pointer names

Use a 2–3 character abbreviation that reflects the domain type, not the word "domain". Use the same
abbreviation for both the local variable in a `FromCR` function and the parameter name in a `ToCR`
function.

| Domain type | Abbreviation |
|-------------|-------------|
| `BlockStorage` | `bs` |
| `Network` | `n` |
| `Workspace` | `ws` |
| `Region` | `r` |
| `StorageSKU` / `NetworkSKU` | `sku` |

### Prefix taxonomy

When two identifiers for different things would otherwise collide in the same scope, apply the smallest
prefix class that resolves the ambiguity:

1. **Kind prefix** — when the same attribute belongs to two different domain objects in scope:
   `blockStorageSizeGB` vs `backupSizeGB`.
2. **Temporal prefix** — when the same attribute is observed at two points in time:
   `lastSizeGB` vs `currentSizeGB`.
3. **Source/target prefix** — when the same attribute appears in two representations:
   `srcRef` vs `dstRef`, `inState` vs `outState`.
4. **Layer prefix** — last resort, only when the type alone does not already disambiguate:
   `crState` vs `domState`. If the types differ (e.g. `schemav1.ResourceState` vs
   `domain.ResourceState`), the type already disambiguates; no prefix is needed.

Never use a layer prefix preemptively — apply only when an actual name collision exists in the scope.

---

## §4 — Receiver naming

Receivers use a 1–3 character abbreviation of the type, consistent across all methods of the type.
Single-letter receivers are preferred for small types; longer abbreviations are fine for larger ones
where a single letter would be ambiguous.

```go
// ✓ correct — consistent abbreviation
func (h *BlockStorageHandler) Create(...) { ... }
func (h *BlockStorageHandler) Delete(...) { ... }

// ✗ wrong — inconsistent or opaque receivers
func (bsh *BlockStorageHandler) Create(...) { ... }
func (handler *BlockStorageHandler) Delete(...) { ... }
```

---

## §5 — Initialisms

The following initialisms are always fully capitalised in hand-written ecp identifiers, regardless of
their position in an exported or unexported name, and regardless of how an adjacent generated or SDK
field capitalises them:

`API` `CR` `CIDR` `GB` `HTTP` `ID` `IOPS` `IP` `IPv4` `IPv6` `SKU` `URL`

```go
// ✓ correct
StorageSKU    PublicIP    NIC    IOPS    CIDR    AdditionalCIDRs

// ✗ wrong (generated/SDK residual — acceptable in zz_generated_* only)
StorageSkuSpec    PublicIpSpec    NicSpec    Iops    Cidr
```

Initialism casing applies to hand-written CR wrapper types (the outer `type X struct { … }`),
function names, variable names, and constants. It does **not** apply to generated `…Spec`/`…Status`
type names, which are controlled by go-sdk. Those are an accepted residual (see Appendix).

---

## §6 — Variable shadowing

Never use a variable name that shadows an imported package alias.

```go
// ✗ wrong — "domain" shadows the imported package alias
func BlockStorageFromAPI(domain *sdk.BlockStorage, id string) *bsdom.BlockStorage {
    return &bsdom.BlockStorage{Spec: bsdom.BlockStorageSpec{ID: domain.Id}}
}

// ✓ correct — typed-short name; no shadow
func BlockStorageFromAPI(bs *sdk.BlockStorage, id string) *bsdom.BlockStorage {
    return &bsdom.BlockStorage{Spec: bsdom.BlockStorageSpec{ID: bs.Id}}
}

// ✗ wrong — "resource" shadows the imported package alias
resource := &schemav1.CommonData{ResourcePath: resourcePath}

// ✓ correct
common := &schemav1.CommonData{ResourcePath: resourcePath}
```

The most common offenders are `domain` (shadows the `domain` import alias) and `resource` (shadows
a slice-local import). Use the typed-short abbreviation (§3) or a more specific name instead.

---

## §7 — Doc comments

Every exported symbol (type, function, constant, variable) must have a doc comment that begins with
the symbol's name.

```go
// ✓ correct
// DefaultPendingCondition is the StatusCondition reported while a resource
// operation is still in progress and no provider-specific condition is available.
var DefaultPendingCondition = StatusCondition{ … }

// ✗ wrong — no doc comment
var DefaultPendingCondition = StatusCondition{ … }
```

Constant re-exports must not introduce stutter. If a slice re-exports a constant from `domain` in its
`backend/kubernetes` package, the re-export name must not repeat the resource name already in the
package path.

```go
// package resource/storage/v1/block-storage/backend/kubernetes
// ✓ correct — "Kind" is enough; the package path already says "block-storage"
const Kind = bsdom.Kind

// ✗ wrong — stutter: the package path already says "block-storage"
const BlockStorageKind = bsdom.Kind
```

---

## §8 — Structural symmetry

Parallel operations on the same domain type must share the same code structure. Two implementations of
the same interface method must look the same: same helpers, same variable names, same error-string
template, same flow.

**Resource-state conversion:** always use the `ResourceStateFromCR` helper from
`resource/common/backend`; never use a raw type cast.

```go
// ✓ correct — uses the shared helper and keeps its chain
state, err := commonbackend.ResourceStateFromCR(cr.Status.State)
if err != nil {
    return nil, fmt.Errorf("block storage %s: %w", cr.Name, err)
}

// ✗ wrong — raw cast bypasses validation and breaks structural symmetry
state := domain.ResourceState(cr.Status.State)
```

**Error strings:** follow the template `"<resource> <name>: <description>: %w"` for all conversion
errors in a given slice, and use the same template across `FromCR`, `ToCR`, `FromAPI`, `ToAPI`.
Drop `<description>` when the wrapped error already carries it — `"block storage bs-1: unknown
resource state \"halfway\""` says everything twice if the caller prefixes "invalid resource
state" as well. **Always carry the name**: `<resource>` alone cannot tell two failing resources
apart in a delegator log that is reconciling hundreds.

**Pending-state predicate:** the `isXPending` helper in every `plugin_handler.go` must apply the same
guard: treat nil status as pending, and only consider deletion pending when `DeletedAt == nil` (i.e.
the resource was not explicitly deleted). See `resource/storage/v1/block-storage/backend/kubernetes/plugin_handler.go`
as the authoritative template.

**Active-state predicate and the update arm:** every `plugin_handler.go` carries an `isXActive`
helper with the same three-part guard — `DeletedAt == nil && Status != nil && State == Active` —
and `HandleReconcile` routes an active resource to `commonbackend.HandleUpdate` before the
create/delete switch. The `DeletedAt` half is what keeps a delete request on an active resource on
the lifecycle path instead of the update one. Where a resource has its own post-active operation,
that operation is checked first (block-storage guards the arm with `!wantBlockStorageIncreaseSize`,
instance runs its power reconcile ahead of it). `HandleUpdate` owns every status write on that arm —
handlers pass it `h.repo` and never call `UpdateStatus` for an update themselves. See
[PLUGINS.md](PLUGINS.md#update-reconciling-an-active-resource) for the contract this implements.

**Trimming conditions is `commonbackend.TrimConditions`,** not a loop re-inlined beside each
`PushCondition`. Every handler that pushes a condition then persists it needs the same bound, and a
trim policy that lives in forty places can only be changed in forty places.

**`commonData.labels` must be sorted.** Every `*ToCR` builds the key list with
`slices.Sorted(maps.Keys(...))`, never `slices.Collect`. This is not cosmetic: the writer adapter
compares stored `commonData` against desired to decide whether an update needs to write at all, and
Go randomises map iteration order — so an unsorted list makes a no-op PUT rewrite the CR, bump its
`resourceVersion`, and trigger a reconcile that hands the plugin a level-triggered `Update` for a
request that changed nothing. Pinned by `TestNetworkToCR_LabelKeysAreSorted` and
`TestWriterAdapter_Update_NoOpDoesNotWrite`.

---

## §9 — Test toolkit

The assertion toolkit is **testify**. Do not introduce a second one — a reader should never have to
learn a new assertion vocabulary to move between two test files.

- `require` is the default. It aborts the test, so a failed precondition cannot cascade into a nil
  dereference in the lines below it.
- `assert` is for independent checks inside a table subtest, where reporting every mismatch in one
  run is more useful than stopping at the first.
- Plain `t.Errorf` is correct in fuzz targets: a fuzz body reports and returns rather than aborting,
  and it must not pull an assertion library into the fuzzing hot path.

Competing toolkits (`gomega`, `ginkgo`, `gotest.tools`, `go-cmp` as an assertion) are denied in test
files by the `test-toolkit` depguard rule in `.golangci.yml`. `go-cmp` remains allowed in non-test
code, where it is a diffing library rather than an assertion library.

---

## §10 — Error contract

There is one error type that crosses a layer boundary: `kernel.Error` (`framework/kernel`). A
caller on the other side of a boundary must be able to `errors.As` an error to `*kernel.Error`,
read its `Kind`, and get an HTTP status or a retry decision out of it without string matching.

### Where `kernel.Error` is required

Wrap with `kernel.NewError(kind, cause, sources…)` at the point where the failure **leaves its
layer** — conversion (`XFromCR`/`XToCR`/`XFromAPI`), persistence adapters, plugin handlers, and
anything a REST handler can reach. Pick the kind by who is at fault:

| Kind | Use for | Status |
|---|---|---|
| `KindValidation` | a value the caller or a stored CR carries that the domain has no representation for | 422 |
| `KindInternal` | a wiring or programming fault — a nil argument, an object type this slice never converts | 500 |
| `KindUnavailable` | an upstream that is not answering — a provider, an informer that will not sync | 500 |
| `KindNotFound` / `KindConflict` / `KindAlreadyExists` / `KindPreconditionFailed` / `KindForbidden` / `KindUnauthorized` | as their names say; `framework/backend/kubernetes` already maps the Kubernetes equivalents | — |

Attach a `kernel.ErrorSource{Name, Value}` when a specific field caused it. It is rendered into
the RFC 7807 response's `sources`, which is what tells a caller *which* field to fix.

### Where a plain error is still correct

A leaf whose only caller wraps it, and startup or CLI errors that are printed to an operator and
never inspected — `gateway/cmd`, `kubeclient`, the `framework/backend/kubernetes/cmd/*` code
generators. Wrapping those buys nothing and costs an import.

### Preserving the chain

- **Always `%w`.** `fmt.Errorf("…: %v", err)` and `errors.New(err.Error())` both sever
  `errors.Is`/`errors.As` at that line, and every layer above loses the kind.
- **Never re-derive an error from a string.** Where a cause genuinely arrives as text — a
  Crossplane `Synced` condition, say — turn it back into an error and wrap *that*, so the chain
  exists even though the original value does not (`csp/ionos/pkg/adapter/crossplane.reconcileError`).
- **Rendering is not wrapping.** `err.Error()` is correct when the destination is a human-readable
  field: a `StatusCondition.Message`, an RFC 7807 `detail`.

### Not swallowing

`_ = f()` is allowed only where the discarded value cannot be acted on, and the line says why in
a comment. Today that is: `hash.Hash.Write` (documented never to fail), a probe response body
written after the status line, a force-close on an already-failing shutdown path, and a poll in a
goroutine nobody joins. Anything else propagates or logs.

The same rule covers enum conversion. Returning the zero value for input you do not recognise is
swallowing with extra steps: it makes a typo indistinguishable from an unset field, and pushes
the rejection to whatever validates next — usually a CRD, which answers the caller with a message
about a field they never wrote. See §2 for which direction rejects and which one trusts.

---

## Appendix — Known residuals

The following known mismatches between this guide and the current codebase are **accepted** and must
not be "fixed" by renaming the generated or SDK artifacts:

| Residual | Reason | Example |
|----------|--------|---------|
| Generated `…SkuSpec`, `…IpSpec`, `NicSpec` casing | go-sdk upstream names them lowercase-tail; `model-gen` copies verbatim; fixing requires an upstream go-sdk change | `NetworkSkuSpec`, `PublicIpSpec`, `NicSpec`, `InstanceSkuSpec` |
| Generated `IpAddress` field | Same upstream reason | `schemav1.IpAddress` |
| region bare group `v1.secapi.cloud` | Intentional: region is the only cluster-scoped global resource; the bare group is hard-coded contract in the shipped CRD, RBAC clusterroles, and ionos/e2e deploy YAML | `resource/region/v1/domain.go` |
| region nested-literal conversion body | Minor shape divergence from slice template; low priority, optional future alignment | `resource/region/v1/frontend/rest/converter.go` |
