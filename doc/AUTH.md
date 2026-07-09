# Authentication & Authorization

This document is the canonical reference for the ECP gateway's auth middleware
chain, introduced in `feat/gateway-auth-middleware`.

---

## Overview

Every incoming HTTP request to the ECP gateway (both global and regional server)
passes through an optional two-stage middleware chain:

```
HTTP request
    │
    ▼
┌─────────────────────────────────────────────┐
│  Authentication middleware                  │
│  reads "Authorization: Bearer <token>"      │
│  validates it, stores *Identity in context  │
└─────────┬───────────────────────────────────┘
          │ success → identity in context
          │ credential failure → 401 Unauthorized
          │ technical failure  → 500 Internal Server Error
          ▼
┌─────────────────────────────────────────────┐
│  Authorization middleware                   │
│  builds AuthorizationClaim from request     │
│  merges Subject + TokenScope into claim     │
│  calls Checker.Authorize(ctx, claim)        │
└─────────┬───────────────────────────────────┘
          │ DecisionAllowed → next handler
          │ DecisionDenied  → 403 Forbidden
          │ DecisionError   → 500 Internal Server Error
          ▼
  provider handler
```

The chain is **opt-in** (default off). Operators enable it per-server with
`--auth-enabled`. Existing deployments are completely unaffected until they set
this flag.

Providers listed in `--authz-skip-providers` (default `seca.region`) install only
the authentication stage: their routes never reach the authorization middleware.
See [Per-provider authorization skip](#per-provider-authorization-skip).

---

## Bearer-Token Format (Dummy Authenticator)

The only authenticator currently shipped is the **Dummy authenticator**, intended
for development and testing. Production deployments will replace it with a real
OIDC/JWT authenticator when that is implemented.

The token is a **Base64-encoded JSON payload**. Only `username` and `password` are
mandatory; the optional `scope` object down-scopes the caller (see below):

```json
{
  "username": "alice",
  "password": "s3cr3t",
  "scope": { "tenants": ["my-tenant"], "regions": ["itbg-bergamo"] }
}
```

Clients send it in the standard HTTP header:
```
Authorization: Bearer <base64-encoded-json>
```

The `username` field becomes `Identity.Subject` after authentication. The
authorization layer matches it against `RoleAssignment.Spec.Subs` to restrict
which role assignments apply to the caller.

**Roles are never carried by the token.** A caller's roles are resolved entirely
from the `Role` and `RoleAssignment` resources in the caller's tenant namespace,
which are managed by the gateway operator. Any `roles` field in the token is ignored.

### Token down-scoping

The optional `scope` object caps what the token may exercise, per SECA scope
dimension (`tenants`, `regions`, `workspaces`). It can only **narrow** the
permissions granted by RBAC — it never grants anything:

- A dimension that is **absent or empty** imposes no restriction.
- A **non-empty** dimension requires the request's tenant/region/workspace to be
  listed, otherwise the request is denied (403).
- A dimension is skipped when the request has no value for it (e.g. `regions` on the
  global server, or `workspaces` on a tenant-level resource).

Down-scoping is useful for issuing narrow tokens (e.g. a CI job limited to one
region) whose blast radius is smaller than the subject's full entitlements.

In code the `scope` object unmarshals into the shared `resource.TokenScope` type
(one definition reused by both the authn and authz ports): it is carried as
`Identity.TokenScope` and copied verbatim into `AuthorizationClaim.TokenScope`.

> ⚠️ **Security caveat**: The Dummy authenticator performs no signature
> verification. Any caller who knows a valid username+password can impersonate
> that subject. It must never be used in production.

---

## Opt-In Configuration

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--auth-enabled` | `false` | Enable bearer-token authn + RBAC authz. |
| `--dummy-auth-users <file>` | `""` | Path to a JSON file mapping `username→password`. Required when `--auth-enabled` is set. |
| `--authz-enabled` | `true` | Install the RBAC authorization middleware. Requires `--auth-enabled`. Set to `false` for authn-only mode (every authenticated caller is let through without a RBAC check). |
| `--authz-skip-providers` | `seca.region` | Comma-separated provider IDs whose routes skip the authorization middleware (authn-only). Neither RBAC nor token down-scoping applies to these providers. |
| `--authz-cache` | `false` | Use the informer-backed `CachedChecker` instead of the per-request `Checker`. |

#### Auth modes

| `--auth-enabled` | `--authz-enabled` | Effect |
|---|---|---|
| `false` | _(irrelevant)_ | No auth. All requests pass through unauthenticated. |
| `true` | `true` (default) | Full authn + authz. Invalid credentials → 401; policy denial → 403. |
| `true` | `false` | Authn-only. Valid credentials → handler; no RBAC check is performed. |

Whatever the mode, a provider listed in `--authz-skip-providers` is served
authn-only even when `--authz-enabled` is true.

### Per-provider authorization skip

SECA RBAC is tenant-scoped: `Role` and `RoleAssignment` objects live in the
tenant's namespace and are evaluated against the tenant/workspace addressed by
the request path. The **region catalog** (`GET /v1/regions[/{name}]`, provider
`seca.region`) is **tenant-less by spec** — upstream confirmed this is the
intended behaviour, not a path shape awaiting correction — so there is no tenant
namespace to load policy from and nothing for RBAC to evaluate.

Providers like this are listed in `--authz-skip-providers` (default:
`seca.region`). Their routes keep the authentication middleware — callers still
need a valid bearer token — but the authorization middleware is never installed
for them:

- **No RBAC check**: any authenticated caller can read the region catalog.
- **No token down-scoping**: the `scope` cap is enforced by the RBAC evaluator,
  so it does not apply to skipped providers either.
- All other providers are unaffected and keep the full authn + authz chain.

The list is configuration, not code: add a provider ID to the flag to move its
routes to the authn-only flow should another catalog-style resource appear.

### Users file format

```json
{
  "alice": "s3cr3t",
  "bob": "p@ssw0rd"
}
```

### Example (development)

```sh
# users.json
echo '{"alice":"s3cr3t"}' > /tmp/users.json

# start the global server with auth enabled
./ecp-gateway globalapiserver \
    --auth-enabled \
    --dummy-auth-users /tmp/users.json

# request with a valid bearer token (roles come from RoleAssignments, not the token);
# the region catalog is authn-only by default (--authz-skip-providers=seca.region)
TOKEN=$(echo '{"username":"alice","password":"s3cr3t"}' | base64 -w0)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/providers/seca.region/v1/regions
```

---

## SECA RBAC Authorization Algorithm

The authorization decision is made by evaluating an `AuthorizationClaim` against
all `Role` and `RoleAssignment` resources in the claim's tenant namespace.

### Algorithm

```
authorized =
    tokenScopeCovers(claim.TokenScope, claim.Tenant, claim.Region, claim.Workspace)
  ∧ ∃ ra ∈ RoleAssignments:
        scopeCovers(ra.Spec.Scopes, claim.Tenant, claim.Region, claim.Workspace)
      ∧ subsGrant(ra.Spec.Subs, claim.Subject)
      ∧ ∃ roleName ∈ ra.Spec.Roles:
            role := rolesByName[roleName]
            ∃ p ∈ role.Spec.Permissions:
                p.Provider == claim.Provider
              ∧ matchResource(p.Resources, claim.Resource, claim.Name)
              ∧ matchVerb(p.Verb, claim.Verb)
```

Roles are taken solely from the matched `RoleAssignment`; the token never carries
roles. `claim.TokenScope` is the optional token cap applied first — a non-empty
dimension must cover the request, or the whole claim is denied.

### Subject matching

`RoleAssignment.Spec.Subs` is a list of JWT subject IDs that this assignment applies to.
The SECA spec makes it **mandatory** (`minItems: 1`). Matching rules:

| Value | Meaning |
|-------|---------|
| `"*"` | Wildcard — covers any authenticated caller. |
| `"user1@example.com"` | Exact match against `claim.Subject`. |
| _(empty list)_ | Grants **nobody** — fail-closed (not a wildcard). |

Unlike scope slices, an empty `Subs` does **not** mean "all subjects". The SECA spec's
explicit `"*"` wildcard design means absence of a subject is always treated as a deny.

### Scope matching

A `RoleAssignmentScope` covers the request when **all three dimensions match**:

| Field | Empty value means |
|-------|-------------------|
| `Tenants` | Wildcard — covers any tenant (within the assignment's namespace). |
| `Regions` | Wildcard — covers any region. |
| `Workspaces` | Wildcard — covers any workspace (or no workspace). |

A non-empty field must contain the claim's value.
The assignment grants access when **at least one scope entry** covers the request.

### Resource matching

`Permission.Resources` is a list of [gobwas/glob](https://github.com/gobwas/glob)
patterns. The match target is:

- Item operation (`claim.Name != ""`): `"<resource>/<name>"` — e.g. `"instances/inst-1"`.
- Collection operation (`claim.Name == ""`): `"<resource>"` — e.g. `"instances"`.

Common pattern examples:

| Pattern | Covers |
|---------|--------|
| `"instances"` | List/collection operations only. |
| `"instances/*"` | Item operations only. |
| `"*"` | Everything (collections and items across all resources). |
| `"networks/subnets"` | Subnet collections. |

### Verb matching

`Permission.Verb` is a list of verb patterns:

| Pattern | Covers |
|---------|--------|
| `"*"` | Any verb. |
| `"get"` | Exact match. |
| `"post"` | Exact `"post"` **and** any sub-action `"post.<action>"` (e.g. `"post.start"`, `"post.restart"`). |
| `"post.start"` | Only `"post.start"` — does not cover `"post.restart"`. |

Verbs are derived from the HTTP method and route pattern:

| HTTP method | Route has `{name}`? | Derived verb |
|-------------|---------------------|--------------|
| GET | No | `list` |
| GET | Yes | `get` |
| PUT | Yes | `put` |
| DELETE | Yes | `delete` |
| POST | After `{name}`, has action segment `<act>` | `post.<act>` |

---

## Error Categories

The auth chain distinguishes three failure categories, each with a distinct HTTP
status and diagnostic handling:

| Category | HTTP | When | How |
|---|---|---|---|
| **Authentication failure** | **401** | Missing, malformed, or invalid bearer token | Middleware writes sanitised `ErrUnauthorized`; real error logged server-side. |
| **Authorization denial** | **403** | Credentials valid but insufficient privileges | Middleware writes sanitised `ErrForbidden`; checker's `DecisionDenied` signals this. |
| **Technical error** | **500** | Infrastructure failure (e.g. RBAC store unreachable) | Middleware logs the detailed error server-side; writes sanitised `ErrInternal`. |

**Important**: technical failures are **never disguised as denials**. A Kubernetes list error in the RBAC checkers produces a `DecisionError` and the middleware responds with HTTP 500, making infrastructure outages immediately visible.

---

## Implementations

### 2.2 — Reader-backed SECA RBAC Checker

`gateway/internal/authz/seca.Checker` (implements `authzport.Checker`).

On every `Authorize` call it:
1. Lists all `RoleAssignment` objects in the claim's tenant namespace via
   `persistence.ReaderRepo[*radom.RoleAssignment]`.
2. Lists all `Role` objects in the same namespace via
   `persistence.ReaderRepo[*roledom.Role]`.
3. Calls the pure `Evaluate` function (no I/O) to determine the decision.

Returns one of three explicit outcomes:
- `DecisionAllowed, nil` — the claim is permitted.
- `DecisionDenied, kernel.ErrForbidden` — policy denies the operation.
- `DecisionError, kernel.KindInternal error` — RBAC data could not be loaded
  (infrastructure failure). The middleware logs the real error and responds HTTP 500.

**Trade-off**: Two Kubernetes API-server round-trips per authorization request.
Suitable for moderate traffic. Use `CachedChecker` for high-throughput paths.

### 2.3 — Informer-backed Cached SECA RBAC Checker

`gateway/internal/authz/seca.CachedChecker` (implements `authzport.Checker`).

Uses `k8s.io/client-go/dynamic/dynamicinformer.DynamicSharedInformerFactory` to
maintain an in-process cache of `Role` and `RoleAssignment` resources, kept
current by Kubernetes watch events. Authorization decisions read from the cache —
zero API-server round-trips on the hot path.

Returns the same three-outcome contract as `Checker` (see above). A cache-read
failure yields `DecisionError` rather than `DecisionDenied`.

**Lifecycle**: `Start(ctx context.Context) error` must be called at server startup
(before serving requests). It pre-registers the informers, starts them, and blocks
until the initial cache sync completes. Pass the server's shutdown context so
informers are stopped on exit.

Enable via `--authz-cache`.

---

## Metrics

Both servers expose a `/metrics` endpoint (unauthenticated, outside the provider
handler chain) that serves the default Prometheus registry in text format.

```
GET /metrics
```

### Auth latency histograms

Three histograms with exponential buckets from ~50 µs to ~3 s are registered
at startup (via `promauto`) regardless of the active checker implementation:

| Metric name | Label | Description |
|-------------|-------|-------------|
| `ecp_gateway_auth_middleware_duration_seconds` | `provider` | End-to-end latency of a single authenticated HTTP request (authn + authz + handler). |
| `ecp_gateway_authz_check_duration_seconds` | `impl` | Latency of one `Checker.Authorize` call, including the RBAC fetch. |
| `ecp_gateway_rbac_fetch_duration_seconds` | `impl` | Latency of the RBAC data fetch inside the checker: `List` from K8s API-server (`impl="direct"`) or in-process informer cache read (`impl="cached"`). |

The `impl` label takes the value `"direct"` when `--authz-cache` is not set, and
`"cached"` when it is. The `provider` label mirrors the registered provider name
(e.g. `"seca.region"`, `"seca.storage"`).

The bucket boundaries are `prometheus.ExponentialBuckets(50e-6, 2, 18)`, giving
18 buckets covering roughly 50 µs → 3 s — wide enough to capture both the
sub-millisecond cached path and the multi-millisecond K8s List path.

The default registry also exposes the standard `go_*` and `process_*` series,
which are useful for comparing allocations and goroutine counts between the two
checker implementations.

---

## Code Layout

```
framework/kernel/port/authn/authn.go       Identity, Authenticator port
framework/kernel/port/authz/authz.go       AuthorizationClaim, Decision, Checker, ClaimExtractor ports
framework/frontend/middleware/
    authentication.go                      NewAuthentication — reads bearer header
    authorization.go                       NewAuthorization — generic authz middleware
    claim.go                               SECAClaimExtractor — derives claim from request
    chain.go                               Chain[M] — typed, order-preserving wrapper
    context.go                             IdentityFromContext

gateway/internal/authn/dummy.go            DummyAuthenticator (dev/test only)
gateway/internal/authz/seca/
    evaluator.go                           Evaluate — pure RBAC evaluation + helpers
    checker.go                             Checker — per-request reader-backed
    cache.go                               CachedChecker — informer-backed
gateway/internal/auth/config.go            Flags, Build, StartChecker, ProviderMWs
gateway/internal/metrics/
    metrics.go                             three histograms, Handler(), Middleware()
    checker.go                             InstrumentedChecker decorator
gateway/cmd/globalapiserver.go             wiring for global providers; /metrics mount
gateway/cmd/regionalapiserver.go           wiring for regional providers; /metrics mount
```

---

## Import Aliases

By convention, packages in this subsystem are aliased as follows:

| Import path | Alias |
|-------------|-------|
| `framework/kernel/port/authn` | `authnport` |
| `framework/kernel/port/authz` | `authzport` |
| `framework/frontend/middleware` | `middleware` |
| `gateway/internal/authn` | `gatewayauthn` |
| `gateway/internal/authz/seca` | `seca` |
| `resource/authorization/v1/role` | `roledom` |
| `resource/authorization/v1/role-assignment` | `radom` |
| `resource/authorization/v1/role/backend/kubernetes` | `rolek8s` |
| `resource/authorization/v1/role-assignment/backend/kubernetes` | `rak8s` |
