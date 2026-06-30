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
└───────────────────┬─────────────────────────┘
                    │ identity in context
                    ▼
┌─────────────────────────────────────────────┐
│  Authorization middleware                   │
│  builds AuthorizationClaim from request     │
│  calls Checker.Authorize(ctx, claim)        │
└───────────────────┬─────────────────────────┘
                    │ nil  → next handler
                    │ err  → 403 Forbidden
                    ▼
            provider handler
```

The chain is **opt-in** (default off). Operators enable it per-server with
`--auth-enabled`. Existing deployments are completely unaffected until they set
this flag.

---

## Bearer-Token Format (Dummy Authenticator)

The only authenticator currently shipped is the **Dummy authenticator**, intended
for development and testing. Production deployments will replace it with a real
OIDC/JWT authenticator when that is implemented.

The token is a **Base64-encoded JSON payload**:

```json
{
  "username": "alice",
  "password": "s3cr3t",
  "roles": ["seca-admin", "compute-viewer"]
}
```

Encoded:
```
eyJ1c2VybmFtZSI6ImFsaWNlIiwicGFzc3dvcmQiOiJzM2NyM3QiLCJyb2xlcyI6WyJzZWNhLWFkbWluIiwiY29tcHV0ZS12aWV3ZXIiXX0=
```

Clients send it in the standard HTTP header:
```
Authorization: Bearer <base64-encoded-json>
```

The `roles` array carries **SECA Role names** (not subjects). These names are
intersected with the `RoleAssignment.Spec.Roles` field during authorization.

> ⚠️ **Security caveat**: The Dummy authenticator performs no signature
> verification. Any caller who knows a valid username+password can claim
> arbitrary roles. It must never be used in production.

---

## Opt-In Configuration

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--auth-enabled` | `false` | Enable bearer-token authn + RBAC authz. |
| `--dummy-auth-users <file>` | `""` | Path to a JSON file mapping `username→password`. Required when `--auth-enabled` is set. |
| `--authz-cache` | `false` | Use the informer-backed `CachedChecker` instead of the per-request `Checker`. |

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

# request with a valid bearer token
TOKEN=$(echo '{"username":"alice","password":"s3cr3t","roles":["seca-admin"]}' | base64 -w0)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/providers/seca.region/v1/tenants/my-tenant/regions
```

---

## SECA RBAC Authorization Algorithm

The authorization decision is made by evaluating an `AuthorizationClaim` against
all `Role` and `RoleAssignment` resources in the claim's tenant namespace.

### Locked algorithm

```
authorized =
    ∃ ra ∈ RoleAssignments:
        scopeCovers(ra.Spec.Scopes, claim.Tenant, claim.Region, claim.Workspace)
      ∧ ∃ roleName ∈ (ra.Spec.Roles ∩ claim.Roles):
            role := rolesByName[roleName]
            ∃ p ∈ role.Spec.Permissions:
                p.Provider == claim.Provider
              ∧ matchResource(p.Resources, claim.Resource, claim.Name)
              ∧ matchVerb(p.Verb, claim.Verb)
```

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

## Implementations

### 2.2 — Reader-backed SECA RBAC Checker

`gateway/internal/authz/seca.Checker` (implements `authzport.Checker`).

On every `Authorize` call it:
1. Lists all `RoleAssignment` objects in the claim's tenant namespace via
   `persistence.ReaderRepo[*radom.RoleAssignment]`.
2. Lists all `Role` objects in the same namespace via
   `persistence.ReaderRepo[*roledom.Role]`.
3. Calls the pure `Evaluate` function (no I/O) to determine the decision.

**Trade-off**: Two Kubernetes API-server round-trips per authorization request.
Suitable for moderate traffic. Use `CachedChecker` for high-throughput paths.

### 2.3 — Informer-backed Cached SECA RBAC Checker

`gateway/internal/authz/seca.CachedChecker` (implements `authzport.Checker`).

Uses `k8s.io/client-go/dynamic/dynamicinformer.DynamicSharedInformerFactory` to
maintain an in-process cache of `Role` and `RoleAssignment` resources, kept
current by Kubernetes watch events. Authorization decisions read from the cache —
zero API-server round-trips on the hot path.

**Lifecycle**: `Start(ctx context.Context) error` must be called at server startup
(before serving requests). It pre-registers the informers, starts them, and blocks
until the initial cache sync completes. Pass the server's shutdown context so
informers are stopped on exit.

Enable via `--authz-cache`.

---

## Code Layout

```
framework/kernel/port/authn/authn.go       Identity, Authenticator port
framework/kernel/port/authz/authz.go       AuthorizationClaim, Checker, ClaimExtractor ports
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
gateway/cmd/globalapiserver.go             wiring for global providers
gateway/cmd/regionalapiserver.go           wiring for regional providers
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
