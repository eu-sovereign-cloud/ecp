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
                      ↖ test/e2e
```

No back-edges. `framework` has zero dependency on `resource`. `resource` has zero dependency on `gateway` or any CSP.

## Resource Model

### Cluster-Scoped Resources

| Resource | Description |
|----------|-------------|
| `Region` | Available regions (read-only) |

Cluster-scoped resources are stored in the `seca` namespace and carry no tenant or workspace qualifier.

### Tenant-Scoped Resources

| Resource | Description |
|----------|-------------|
| `Workspace` | Logical grouping of resources within a tenant |
| `BlockStorage` | Block storage volume |
| `Network` | Network resource |
| `StorageSKU` / `NetworkSKU` | Available SKU options (read-only) |

### Namespacing Strategy

- The `seca` namespace groups cluster-scoped and shared resources.
- Each `Tenant` CR triggers the creation of a dedicated tenant namespace; all tenant-scoped resources owned by that tenant live there.
- `Workspace` CRs are placed in the tenant namespace and labeled with their parent tenant.

## Authentication & Authorization

The gateway enforces an opt-in bearer-token authn + SECA RBAC authz middleware
chain. When enabled (`--auth-enabled`), every request must carry a valid
`Authorization: Bearer <token>` header and the decoded identity must be
authorised by the RBAC policy before the request reaches the handler.

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

All framework-layer types (`Authenticator`, `Checker`, `ClaimExtractor`,
`AuthorizationClaim`) live under `framework/kernel/port/{authn,authz}` and are
resource-agnostic. Concrete implementations (`DummyAuthenticator`, SECA RBAC
`Checker`, `CachedChecker`) live in `gateway/` and may import `resource/`.

See [doc/AUTH.md](AUTH.md) for the full reference — bearer-token format, config
flags, the locked RBAC algorithm, and a code layout map.

## Cascaded Deletion

ECP enforces owner-reference–based cascaded deletion:

- Deleting a **Tenant** cascades to all its Workspaces and all resources within them.
- Deleting a **Workspace** cascades to all resources within that workspace.
