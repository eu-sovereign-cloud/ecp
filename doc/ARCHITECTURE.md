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

There is no `Tenant` CRD. Namespaces are derived deterministically from the resource's SECA scope (`framework/backend/kubernetes/adapter.go`), by one of two hash formulas selected in `resolveNamespace`:

- `ComputeNamespace` — `sha3-224(<tenant>)` for tenant-scoped resources, `sha3-224(<tenant>/<workspace>)` for workspace-scoped ones.
- `ComputeNetworkNamespace` — `sha3-224(<tenant>/<workspace>/<network>)` for network-scoped resources (`Subnet`, `RouteTable`), so each network gets its own namespace and its children's names only have to be unique per network.

An empty scope yields no namespace — that is the cluster-scoped `Region` case.

Three levels of namespace exist, and each is labeled with the internal `secapi.cloud/{tenant,workspace,network}` owner labels that identify who provisioned it:

| namespace | holds |
|---|---|
| `sha3-224(tenant)` | `Workspace`, `Role`, `RoleAssignment`, `Image`, SKU catalogs |
| `sha3-224(tenant/workspace)` | `BlockStorage`, `Network`, `NIC`, `PublicIP`, `InternetGateway`, `SecurityGroup`, `SecurityGroupRule`, `Instance` |
| `sha3-224(tenant/workspace/network)` | `Subnet`, `RouteTable` |

### Namespace lifecycle

Creation is split between the write path and the delegator. `NamespaceManagingWriterAdapter.Create` provisions the tenant namespace *before* the CR, because the CR itself lives in it and the write would otherwise fail with `NotFound`. The namespace an entity owns for its **children** is provisioned *after* the CR and opportunistically: nothing is waiting on it inside that request, so its failure is logged rather than returned and the create still succeeds.

Only the **tenant** namespace is provisioned on a resource's own behalf, because there is no `Tenant` entity that would otherwise create it. Below that level a namespace is owned by a parent entity and its absence *is* the referential-integrity check: a `Network` addressed to a workspace that was never created fails with `NotFound` rather than fabricating `sha3-224(tenant/workspace)` and stranding resources in a workspace no listing will ever return. The same holds for every leaf resource, which uses a plain `WriterAdapter` and never creates a namespace at all.

Only entities that own a namespace go through that adapter (`Workspace`, `Network`, and `Role`/`RoleAssignment` on the global gateway). Ordering the child namespace after the CR is what makes it recoverable: the CR is the namespace's only owner, so one created first and then orphaned by a failed CR write leaks permanently, while a CR written without its namespace is repaired on the next reconcile. The tenant namespace needs no such care — it is shared, bounded by the authenticated tenant, and adopted by the next create.

The owning controller in the delegator is the backstop, via the `GenericController.WithEnsure` hook: `NamespaceEnsure` runs before the plugin handler on every reconcile of a live resource, and gates it — a failure requeues with backoff instead of letting the resource go `active` without its namespace. That closes the crash window completely — a gateway that dies after writing the CR leaves a *new* CR, which always reconciles — and it is what repairs a namespace deleted or relabelled out of band, which nothing else ever did. The second case is prompt only if something reconciles the CR: the manager sets no `SyncPeriod`, so a settled resource is otherwise resynced on controller-runtime's default period. `CreateNamespace` is idempotent and stamps the owner labels teardown checks, so a repaired namespace stays reclaimable. The hook is skipped once the resource is being deleted, so it cannot race `NamespaceCleanup` back into existence.

The child namespace is therefore eventually consistent while the CR is not: a child resource created in the window between the two fails with `NotFound` and has to be retried by its caller.

Teardown belongs to the owning controller in the delegator, via the `GenericController.WithCleanup` hook: `NamespaceCleanup` runs once, after the plugin has finished deleting and before the finalizer is dropped. It re-checks emptiness, verifies ownership, then deletes the namespace. Because the finalizer is still held, a failure is retried instead of orphaning the namespace. A namespace whose owner labels do not match is left in place and logged — deleting someone else's namespace is worse than leaking one.

The lists of types that may live in each child namespace are `ChildResourceGVRs`, exported by the owning slice (`resource/workspace/v1/backend/kubernetes` and `resource/network/v1/network/backend/kubernetes`) and shared by the gateway's 409 check and the controller's re-check. A type missing from a list makes its namespace look empty when it is not.

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

The SECA resource organization is hierarchical — Tenants 1—\* Workspaces 1—\* Networks 1—\* resources — and deletion is intended to cascade down this hierarchy. The building block for this is namespace ownership rather than Kubernetes owner references (none are set today):

- A `Workspace`'s resources live in the workspace's dedicated namespace, and a `Network`'s in its own (see [Namespacing Strategy](#namespacing-strategy)), so deleting that namespace removes everything under it at once.
- The refusal and the teardown are split. `NamespaceManagingWriterAdapter.Delete` refuses the delete with 409 while the child namespace still holds SECA resources — a user-facing invariant that has to answer synchronously — and then deletes only the CR. Tearing the namespace down is the owning controller's `WithCleanup` finalizer, which re-checks emptiness before it acts.
- With no `Tenant` entity there is no tenant-level deletion to cascade from.

An owner reference cannot substitute for the finalizer here: `Namespace` is cluster-scoped while `Workspace` and `Network` are namespaced, and Kubernetes treats a cluster-scoped dependent with a namespaced owner as an unresolvable reference — the namespace is never garbage collected and the GC emits recurring `OwnerRefInvalidNamespace` events (before k8s 1.20 it deleted the dependent instead). Only a cluster-scoped owner would work, and there is no such object today.
