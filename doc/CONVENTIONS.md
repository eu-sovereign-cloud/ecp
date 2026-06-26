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
| API → domain | `XFromAPI(sdk …, id, region string) *dom.X` |
| domain → API | `XToAPI(x *dom.X) *sdk.X` |
| domain list → API list | `XIteratorToAPI(iter …) []sdk.X` |
| domain → API with HTTP verb | `XToAPIWithVerb(x *dom.X, verb string) *sdk.X` |

### Sub-object helpers in `resource/common`

The same `FromCR`/`ToCR`/`FromAPI`/`ToAPI` suffixes apply to sub-object converters:

```
ReferenceFromCR / ReferenceToCR / ReferenceFromAPI / ReferenceToAPI / ReferencePtrToAPI
StatusConditionFromCR / StatusConditionToCR
ResourceStateFromCR / ResourceStateToCR / ResourceStateToAPI
ConditionsToAPI / conditionToAPI   (conditionToAPI is unexported — lower-case is intentional)
```

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
// ✓ correct — uses the shared helper
state, err := commonbackend.ResourceStateFromCR(cr.Status.State)
if err != nil {
    return nil, fmt.Errorf("block storage %s: invalid resource state: %w", cr.Name, err)
}

// ✗ wrong — raw cast bypasses validation and breaks structural symmetry
state := domain.ResourceState(cr.Status.State)
```

**Error strings:** follow the template `"<resource> <name>: <description>: %w"` for all conversion
errors in a given slice. Use the same template across `FromCR`, `ToCR`, `FromAPI`, `ToAPI`.

**Pending-state predicate:** the `isXPending` helper in every `plugin_handler.go` must apply the same
guard: treat nil status as pending, and only consider deletion pending when `DeletedAt == nil` (i.e.
the resource was not explicitly deleted). See `resource/storage/v1/block-storage/backend/kubernetes/plugin_handler.go`
as the authoritative template.

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
