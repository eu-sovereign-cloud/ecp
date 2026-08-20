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
│  merges identity (subject + caps) into it   │
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

## Bearer-Token Format

Two authentication plugins ship today, selected with `--auth-plugin`:

| `--auth-plugin` | Token | Use |
|---|---|---|
| `dummy` (default) | Base64-encoded JSON with `username` + `password` | Development and testing only — **no signature verification**. |
| `jwt` | A standard signed JWT (compact `header.payload.signature`) | Verifies the signature against a configured key; the shape a real issuer produces. |

Whichever plugin is active, the rest of the chain is identical: the authenticator
produces an `Identity` carrying a **subject** and an optional **token scope**, and
the authorization layer consumes only those. The two sections below cover just the
token each plugin accepts; [Token down-scoping](#token-down-scoping) and everything
after it apply to both.

### Dummy authenticator (`--auth-plugin=dummy`)

The token is a **Base64-encoded JSON payload**. Only `username` and `password` are
mandatory; the optional `scope` object down-scopes the caller and the optional `tenants`
list stands in for the membership a real issuer would stamp (see below):

```json
{
  "username": "alice",
  "password": "s3cr3t",
  "tenants": ["my-tenant"],
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

### JWT authenticator (`--auth-plugin=jwt`)

The token is a **standard signed JWT** in compact form, sent verbatim — no extra
encoding wraps it:

```
Authorization: Bearer <header>.<payload>.<signature>
```

The payload uses registered claims plus the same optional `scope` object:

```json
{
  "sub": "alice",
  "exp": 1893456000,
  "iss": "https://idp.example",
  "aud": "seca-api",
  "tenants": ["my-tenant"],
  "scope": { "tenants": ["my-tenant"], "regions": ["itbg-bergamo"] }
}
```

| Claim | Required | Meaning |
|-------|----------|---------|
| `sub` | yes | Becomes `Identity.Subject` — the same role of the dummy token's `username`. A token without it is rejected. The subject claim is `sub`, platform-wide: it is what `RoleAssignment.Spec.Subs` lists, so an issuer that identifies users by email must put the email in `sub`. |
| `exp` | yes | Expiry. Enforced with `jwt.WithExpirationRequired`, so a token that never expires is rejected rather than honoured forever. |
| `iss` | when `--jwt-issuer` is set | Must equal the configured issuer. Setting the flag also makes the claim mandatory. |
| `aud` | when `--jwt-audience` is set | Must contain the configured audience. Setting the flag also makes the claim mandatory. |
| `tenants` | no | Issuer-asserted tenant membership — see [Tenant membership](#tenant-membership-the-tenants-claim). |
| `scope` | no | Token down-scope, identical to the dummy token's (see below). |

A token is accepted only when its signature verifies **and** its `alg` header
matches `--jwt-signing-method` exactly (`jwt.WithValidMethods`). Pinning the
algorithm is what defeats *algorithm confusion*: without it, an attacker could
re-sign a token as `HS256` using the gateway's own public key as the HMAC secret
and have it verify. Any algorithm `golang-jwt` supports may be configured.

As with the dummy plugin, **roles are never carried by the token** — a `roles`
claim is ignored, and entitlements come only from RoleAssignments.

#### Issuer and audience (`--jwt-issuer`, `--jwt-audience`)

A valid signature only says *this key signed it*. It does not say who minted the
token or which service it was minted for, and the same key routinely signs tokens
for other audiences. Both checks are therefore configuration, not claims to be
trusted from the token itself:

- `--jwt-issuer` — the expected `iss`. A token from another issuer sharing the key
  is rejected.
- `--jwt-audience` — the expected `aud` (matched against the claim's list). A token
  minted for a different service and replayed here is rejected.

Each is enforced only when set, and setting it also makes that claim **mandatory** —
a token that simply omits `iss`/`aud` is rejected exactly like one carrying the wrong
value, so an issuer cannot opt out of the check by dropping the claim. Leave them
empty only when a single issuer mints tokens for this gateway alone.

Nothing named *inside* the token selects how it is verified: the key, the signing
method, the issuer and the audience all come from flags. A token that names its own
verification endpoint is verifying itself.

#### Verification key (`--jwt-secret`)

`--jwt-secret` is a **path to a file**, not the key itself: a key belongs in a
mounted volume (the same pattern as TLS certs and kubeconfigs), and passing it as a
flag value would leak it into `ps` output, shell history, and pod specs. Because the
gateway only opens a path, a Secret and a ConfigMap mount identically — no code
depends on which you choose.

**Which to choose is a security decision, not a preference:**

| Method | Key is | Store in |
|---|---|---|
| `HS256` / `HS384` / `HS512` | the **shared HMAC secret** — it *signs* as well as verifies | a **Secret**. Anyone who can read it can mint a token for any subject, so a ConfigMap here is an auth bypass: ConfigMaps are readable by anything with `get configmap` in the namespace and are not encrypted at rest. |
| `ES*` / `RS*` / `PS*` / `EdDSA` | a **public key** — verification only, the private half never reaches the gateway | either. A public key is not confidential, so a ConfigMap is fine and idiomatic; a Secret is harmless. |

Asymmetric methods are the better default for exactly this reason: the gateway
cannot mint tokens even if fully compromised. Prefer them over `HS*` unless a
shared secret is forced on you.

What the file must contain also depends on the method, because `golang-jwt` requires
the key as a typed Go value rather than raw bytes:

| `--jwt-signing-method` | File content | Parsed to |
|---|---|---|
| `HS256` / `HS384` / `HS512` | The raw HMAC secret, used verbatim | `[]byte` |
| everything else (`ES*`, `RS*`, `PS*`, `EdDSA`) | A PEM-encoded PKIX public key (`-----BEGIN PUBLIC KEY-----`) | `*ecdsa.PublicKey`, `*rsa.PublicKey`, `ed25519.PublicKey` |

`gatewayauthn.ParseVerifyKey` does this conversion at startup: one
`x509.ParsePKIXPublicKey` call returns whichever concrete type the method needs, so
no per-algorithm branching is involved. An unreadable file, an unparseable key, or
an unknown signing method **fails the server at startup** rather than rejecting
every request at runtime.

> Generate a key pair with:
> ```sh
> openssl ecparam -name prime256v1 -genkey -noout -out jwt-key.pem   # ES256 private key
> openssl ec -in jwt-key.pem -pubout -out jwt-key.pub                # public key for --jwt-secret
> ```

#### Token lifetime and revocation

Verification is **offline** — one signature check against a key the gateway already holds
— so a token stays valid until it expires whatever happens to the account behind it. Use
**short-lived tokens plus refresh at the issuer**: revocation latency is whatever TTL your
issuer mints, and `exp` is mandatory (`jwt.WithExpirationRequired`) so a token cannot opt
out. Removing the caller's `RoleAssignment` is immediate by comparison — RBAC is read per
request. Per-request introspection is deliberately not implemented
([AUTH-SPEC-REVIEW.md](AUTH-SPEC-REVIEW.md) §6).

### Tenant membership (the `tenants` claim)

The optional `tenants` claim is the tenant membership the **issuer** asserts about the
subject — static onboarding data the CSP's IdP genuinely owns (see
[AUTH-SPEC-REVIEW.md](AUTH-SPEC-REVIEW.md) §1). It reaches the authorization layer as
`Identity.MemberTenants` and gates every request: when the list is non-empty, the
request's tenant must appear in it or the request is denied with **403**.

It looks like `scope.tenants` and matches identically, but the two differ in who asserts
them, which is the whole point:

| | `tenants` | `scope.tenants` |
|---|---|---|
| Asserted by | the issuer, on every token it mints | the caller, when requesting the token |
| Can be omitted by the caller | no | yes |
| Caps a `subs: ["*"]` assignment | yes | no — the caller just omits it |

Both are enforced, so the effective cap is their intersection: an issuer-stamped
membership of `["a","b"]` with a self-requested scope of `["b"]` reaches only `b`.

The claim is a **gate, not a grant**: being a member of a tenant authorizes nothing on
its own — a RoleAssignment in that tenant's namespace still has to grant the operation.
An absent claim imposes no gate, which is what keeps pre-existing issuers (and every
fixture token) working; deploy an issuer that always stamps it if you rely on the gate.
Like the down-scope, it is skipped on a request that carries no tenant at all — the only
such path today is the region catalog, which is served authn-only anyway.

> The claim name is fixed (`tenants`). An IdP that names it otherwise (`tid`, `org_id`, …)
> needs a claim mapping on the issuer side.

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
| `--auth-plugin` | `dummy` | Authenticator to install: `dummy` or `jwt`. |
| `--dummy-auth-users <file>` | `""` | Path to a JSON file mapping `username→password`. Required when `--auth-plugin=dummy`. |
| `--jwt-signing-method` | `ES256` | Expected JWT `alg`; tokens signed with anything else are rejected. Any `golang-jwt` method is accepted. Required when `--auth-plugin=jwt`. |
| `--jwt-secret <file>` | `""` | Path to the verification key file: the raw HMAC secret for `HS*`, a PEM public key otherwise. Required when `--auth-plugin=jwt`. |
| `--jwt-issuer` | `""` | Expected `iss`. Enforced (and required in the token) only when set. |
| `--jwt-audience` | `""` | Expected `aud`. Enforced (and required in the token) only when set. |
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

### Example: dummy plugin (development)

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

### Example: jwt plugin

```sh
# an ES256 key pair; the server only ever sees the public half
openssl ecparam -name prime256v1 -genkey -noout -out /tmp/jwt-key.pem
openssl ec -in /tmp/jwt-key.pem -pubout -out /tmp/jwt-key.pub

./ecp-gateway globalapiserver \
    --auth-enabled \
    --auth-plugin jwt \
    --jwt-signing-method ES256 \
    --jwt-secret /tmp/jwt-key.pub \
    --jwt-issuer https://idp.example \
    --jwt-audience seca-api

# mint a token with your issuer (or the e2e helper, authhelper.SignJWT) and send it as-is
curl -H "Authorization: Bearer $JWT" http://localhost:8080/providers/seca.region/v1/regions
```

---

## SECA RBAC Authorization Algorithm

The authorization decision is made by evaluating an `AuthorizationClaim` against
all `Role` and `RoleAssignment` resources in the claim's tenant namespace.

### Algorithm

```
authorized =
    tokenScopeCovers(claim.MemberTenants, claim.Tenant)
  ∧ tokenScopeCovers(claim.TokenScope, claim.Tenant, claim.Region, claim.Workspace)
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
roles. `claim.MemberTenants` (issuer-asserted membership) and `claim.TokenScope`
(the caller's down-scope) are the optional caps applied first — a non-empty one must
cover the request, or the whole claim is denied. Being applied before the assignment
loop is what lets the membership gate constrain a `subs: ["*"]` grant.

### Subject matching

`RoleAssignment.Spec.Subs` is a list of JWT subject IDs that this assignment applies to.
The SECA spec makes it **mandatory** (`minItems: 1`). Matching rules:

| Value | Meaning |
|-------|---------|
| `"*"` | Wildcard — covers any authenticated caller, narrowed to the token's `tenants` membership when the issuer stamps one. |
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
gateway/internal/authn/jwtstd.go           JwtAuthenticator + ParseVerifyKey (key file → typed key)
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
