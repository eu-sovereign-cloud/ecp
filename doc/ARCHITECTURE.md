# ECP Architecture

This document describes the design and implementation of the ECP (European Control Plane).

## Overview

The ECP is a distributed control plane for managing and orchestrating cloud resources across multiple cloud service providers (CSPs). It exposes a unified, declarative REST API; all managed resources are persisted as Kubernetes Custom Resources (CRs), providing compatibility with existing Kubernetes tooling and GitOps workflows.

## Two-Axis Module Topology

The repo is organized around two orthogonal axes, each a separate Go module:

```
              framework/                   (module …/ecp/framework)
              ├─ kernel             ← leaf: ALL abstractions (ports, Scope, Error, validation)
              ├─ backend/kubernetes → kernel: k8s adapter, schema/v1 CRDs, codegen, GenericController, ControllerSet
              └─ frontend           → kernel: httpserver, kubeclient, logger, config
                    │
                    ▼  framework ↛ resource (COMPILER-ENFORCED module boundary)
              resource/                    (module …/ecp/resource)
               ├─ common/{domain,frontend,backend}   shared backbone
               └─ <group>/vN/<resource>/
                   ├─ domain.go        canonical type + identity consts (package <resource>)
                   ├─ frontend/rest/   REST↔domain converters + HTTP handlers (per-group, shared handler)
                   └─ backend/kubernetes/ CR types, adapters, controller, plugin iface+handler
                         │
              ┌──────────┴──────────┐
           gateway/             csp/{dummy,ionos,aruba}/
      (→ framework, resource)   (→ framework, resource)
```

**Horizontal axis** (`framework/`): the architectural *layers* — generic, resource-agnostic, shared by everything. Change a layer once and it applies to all resources.

**Vertical axis** (`resource/`): the *features* — one self-contained bounded context per resource, cutting through all layers. Change a resource in one place; nothing else needs editing.

**Module boundary**: `framework ↛ resource` is enforced by the Go compiler (separate modules). A `framework` package that imports `resource` fails to build under `GOWORK=off`. This is the repo's load-bearing invariant.

## Layer DAG (within framework/)

Inter-layer ordering is enforced by `depguard` in `.golangci.yml`:

```
kernel             — pure leaf (stdlib + gobwas/glob only)
backend/kubernetes → kernel
frontend           → kernel
```

## Per-Resource Slice (vertical hexagon)

Each resource slice at `resource/{group}/vN/{resource}/` contains:

- **`domain.go`** (`package <resource>`) — the canonical domain type, `RegionalMetadata` embed, and identity consts (`Kind`, `Resource`, `Group`, `Version`, and a provider identifier). No k8s imports.
- **`frontend/rest/`** — REST↔domain converters and, for the group owner, HTTP handlers implementing the go-sdk `ServerInterface`. One handler per API group (shared across sibling resources); per-resource files are `<resource>_handler.go` and `<resource>_converter.go`. Registered into the gateway mux.
- **`backend/kubernetes/`** — CR wrapper types, GVR/GVK, CR↔domain adapter (`conversion.go`), plugin interface (`plugin.go`), plugin handler (`plugin_handler.go`), and controller wiring (`controller.go`). The `NewController` factory performs **builder inversion**: it assembles the `framework/backend/kubernetes` repo adapter from this slice's own GVR and mappers, wraps it in `framework/backend/kubernetes/controller.GenericController[D]`, and returns a `framework/backend/kubernetes/builder.Reconciler` — no `framework` package ever names a concrete resource.

## Module DAG

```
framework   ← resource ← gateway
                      ↖ csp/{dummy,ionos,aruba}
                      ↖ test
```

No back-edges. `framework` has zero dependency on `resource`. `resource` has zero dependency on `gateway` or any CSP.

## Resource Model

The control plane manages 18 resource slices — one CRD each, generated into `charts/ecp/crds/` (see [CODEGEN.md](CODEGEN.md)) — organized by SECA API group:

| API group | Resources |
|-----------|-----------|
| `v1.secapi.cloud` (`seca.region`) | `Region` — region catalog (read-only, cluster-scoped) |
| `workspace.v1.secapi.cloud` | `Workspace` — logical grouping of resources within a tenant |
| `authorization.v1.secapi.cloud` | `Role`, `RoleAssignment` — SECA RBAC policy (see [AUTH.md](AUTH.md)) |
| `storage.v1.secapi.cloud` | `BlockStorage`, `Image`, `StorageSKU` (read-only catalog) |
| `network.v1.secapi.cloud` | `Network`, `Subnet`, `NIC`, `PublicIP`, `RouteTable`, `InternetGateway`, `SecurityGroup`, `SecurityGroupRule`, `NetworkSKU` (read-only catalog) |
| `compute.v1.secapi.cloud` | `Instance`, `InstanceSKU` (read-only catalog) |

`Region` is the only cluster-scoped CRD: it carries no tenant or workspace qualifier (tenant-less by spec — the gateway serves it authn-only, see [AUTH.md](AUTH.md)). Every other resource is namespaced.

### Namespacing Strategy

There is no `Tenant` CRD. Namespaces are derived deterministically from the resource's SECA scope by `ComputeNamespace` (`framework/backend/kubernetes/adapter.go`): the SHA3-224 hash of `<tenant>` for tenant-scoped resources, or of `<tenant>/<workspace>` for workspace-scoped ones.

- Tenant-scoped resources (`Workspace`, `Role`, `RoleAssignment`, SKU catalogs) live in the tenant namespace `sha3-224(tenant)`.
- Creating a `Workspace` also creates the namespace `sha3-224(tenant/workspace)` that holds the workspace's resources (e.g. `BlockStorage`), labeled with internal tenant/workspace owner labels; the namespace is rolled back if the workspace create fails.
- An empty scope yields no namespace — that is the cluster-scoped `Region` case.

## Authentication & Authorization

The gateway enforces an opt-in bearer-token authn + SECA RBAC authz middleware
chain. When enabled (`--auth-enabled`), every request must carry a valid
`Authorization: Bearer <token>` header and the decoded identity must be
authorised by the RBAC policy before the request reaches the handler. A caller's
roles are resolved from `RoleAssignment`/`Role` in the tenant namespace — never
from the token, which carries only the subject and an optional down-scope.

```
HTTP request
    │
    ▼
NewAuthentication  — validates bearer token → Identity in context (401 on failure)
    │
    ▼
NewAuthorization   — builds AuthorizationClaim, calls Checker.Authorize (403 on denial)
    │
    ▼
provider handler
```

The middleware chain is assembled once at startup in `gateway/internal/auth/config.go`
via `ProviderMWs[M]`, which returns the correctly reversed slice required by
oapi-codegen's `StdHTTPServerOptions.Middlewares`.

Not every provider gets both stages: providers listed in `--authz-skip-providers`
(default `seca.region` — the region catalog is tenant-less by spec, so tenant-scoped
RBAC cannot govern it) are served authn-only, i.e. the authorization middleware is
never installed for their routes.

All framework-layer types (`Authenticator`, `Checker`, `ClaimExtractor`,
`AuthorizationClaim`) live under `framework/kernel/port/{authn,authz}` and are
resource-agnostic. Concrete implementations (`DummyAuthenticator`, SECA RBAC
`Checker`, `CachedChecker`) live in `gateway/` and may import `resource/`.

See [doc/AUTH.md](AUTH.md) for the full reference — bearer-token format, token
down-scoping, config flags, the RBAC algorithm, and a code layout map.

## Cascaded Deletion

The SECA resource organization is hierarchical — Tenants 1—\* Workspaces 1—\* resources — and deletion is intended to cascade down this hierarchy. The building block for this is namespace ownership rather than Kubernetes owner references (none are set today):

- A `Workspace`'s resources live in the workspace's dedicated namespace (see [Namespacing Strategy](#namespacing-strategy)), so deleting that namespace removes everything in the workspace at once.
- `NamespaceManagingWriterAdapter.Delete` refuses delete when the child namespace still has SECA resources (empty check over injected GVRs), then deletes the CR and the owned child namespace. With no `Tenant` entity there is no tenant-level deletion to cascade from.
