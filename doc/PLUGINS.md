# Plugin System

This document describes the ECP plugin architecture, the available CSP plugins, and how to implement a new one.

## Overview

CSP plugins implement the resource interfaces defined in each resource slice's `backend/kubernetes/plugin.go` and are called by the per-slice Kubernetes controllers to perform the actual provisioning and management of cloud resources.

Each plugin is a separate Go module under `csp/`, keeping CSP-specific dependencies isolated from the rest of the codebase.

## Plugin Interfaces

Plugin interfaces live in each resource slice at `resource/{group}/vN/{resource}/backend/kubernetes/plugin.go`. This co-locates the interface with the controller and handler that use it — no framework package ever names a concrete resource.

Every interface has `Create`, `Delete` and `Update`. Some add operations of their own for state a resource carries beyond its lifecycle — `IncreaseSize` on a volume, `PowerOn`/`PowerOff` on an instance.

**`WorkspacePlugin` interface** (`resource/workspace/v1/backend/kubernetes/plugin.go`):
```go
type WorkspacePlugin interface {
    Create(ctx context.Context, resource *wsdom.Workspace) error
    Delete(ctx context.Context, resource *wsdom.Workspace) error
    Update(ctx context.Context, resource *wsdom.Workspace) error
}
```

**`BlockStoragePlugin` interface** (`resource/storage/v1/block-storage/backend/kubernetes/plugin.go`):
```go
type BlockStoragePlugin interface {
    Create(ctx context.Context, resource *bsdom.BlockStorage) error
    Delete(ctx context.Context, resource *bsdom.BlockStorage) error
    Update(ctx context.Context, resource *bsdom.BlockStorage) error
    IncreaseSize(ctx context.Context, resource *bsdom.BlockStorage) error
}
```

**`NetworkPlugin` interface** (`resource/network/v1/network/backend/kubernetes/plugin.go`):
```go
type NetworkPlugin interface {
    Create(ctx context.Context, resource *netdom.Network) error
    Delete(ctx context.Context, resource *netdom.Network) error
    Update(ctx context.Context, resource *netdom.Network) error
}
```

A CSP plugin implements these interfaces for each resource type it supports.

## Update: reconciling an active resource

`Create` and `Delete` are **edge-triggered** — the reconciler calls them once per lifecycle transition, on the edge into `creating` or `deleting`. `Update` is **level-triggered**: once a resource is active it has no transition left to make, so it is handed to `Update` on *every* reconcile, and the plugin decides for itself whether anything needs doing.

This is why a plugin must:

- **Be idempotent, and cheap when nothing changed.** Compare the desired state against the provider and return `nil` if they already agree. Do not write unconditionally — the controller reconciles on its own writes, so a write per pass never settles.
- **Not assume it is told what changed.** Nothing diffs the resource for you. There is no observed copy of the spec to compare against, because the SECA spec publishes observed state for almost nothing (`status.sizeGB` and `status.powerState` are the exceptions, which is exactly why `IncreaseSize` and `PowerOn`/`PowerOff` can be edge-triggered and a generic update cannot).

Where a resource has both, its own operation wins: a pending resize is routed to `IncreaseSize`, and a pending power change to `PowerOn`/`PowerOff`, before `Update` is considered.

### Reporting the outcome

| Return | Meaning | Reconciler does |
|---|---|---|
| `nil` | applied, or nothing to apply | clears any previous `UpdateFailed`; writes no status otherwise |
| `backend.StillProcessing` | in flight | requeues at the controller's interval, leaves status untouched |
| `backend.Revisit(d)` | in flight, come back after `d` | requeues after `d`, leaves status untouched |
| `backend.RevisitBecause(d, cause)` | in flight; `cause` explains the wait | requeues after `d`; `cause` reaches the log |
| error wrapping `backend.ErrNotSupported` | the provider will never accept this | records the reason, **does not retry** |
| any other error | assumed transient | records the reason and requeues at the controller's configured interval |

`StillProcessing`, `Revisit` and `RevisitBecause` are **progress signals, not failures**. They ride the error channel because that is the only channel every frame in a plugin's call chain has — the same reason `io.EOF` does — and the reconciler treats them as a reschedule, not a fault. A zero duration means the controller's configured interval, never "immediately".

`HandleUpdate` converts ordinary update failures to `RevisitBecause(0, cause)`, preserving the controller's configured retry cadence while carrying the cause to status and logs. Outside this update path, return a plain error when exponential backoff is required.

A progress signal must be the **outermost** error. Never wrap a failure inside one: the reconciler classifies failures first, so a wrapped `ErrNotSupported` still stops, but anything else you hide in there becomes a quiet reschedule. Don't build one with `fmt.Errorf("%w: ...", backend.StillProcessing)` either — that just re-hides a failure the same way. If a progress signal needs to carry a cause, use `backend.RevisitBecause(d, cause)`; it is already the outermost error and the cause is only ever a log explanation, never something the reconciler must classify.

`ErrNotSupported`, by contrast, *is* a failure, and it is the plugin port's own vocabulary for one — it's matched with `errors.Is`, so wrap it with `%w` rather than re-wording it. Everything else a plugin returns follows the
repo-wide contract in [CONVENTIONS.md §10](CONVENTIONS.md#10--error-contract): wrap the cause, and
use `kernel.NewError` where the failure leaves the plugin — the reason ends up verbatim in the
resource's condition, so a stringified chain is a reason the operator cannot act on.

`ErrNotSupported` is for a change the provider cannot make at all, not one that has not finished. Cloud resources routinely have immutable fields — an Aruba VPC's region, an instance's flavor — and retrying those re-issues a request the provider has already refused. Wrap it so the reason reaches the user:

```go
return fmt.Errorf("%w: an Aruba VPC's region cannot be changed after creation", backend.ErrNotSupported)
```

**Only return it for a diff you have actually detected.** Because `Update` runs on every reconcile and not only after an edit, an unconditional `ErrNotSupported` reports a refusal on resources nobody touched — every one of them, forever. That is worse than saying nothing: it destroys the condition's signal value, since a reader can no longer tell "nothing to do" from "refused". A plugin that cannot diff at all should return `nil` and document the gap (see the IONOS plugin below).

A failed update leaves the resource **active**, with an `UpdateFailed` condition carrying the message. It is still running and healthy; it just no longer matches its spec. Holding it active is also what keeps the failure recoverable — `error` matches no arm of the reconciler, so the resource would be stranded there and a corrected spec would never be retried.

The condition is retracted as soon as an update succeeds — including when it has since been buried under later conditions, which is the normal case for a resource that also has its own post-active operation (a resize, a power transition).

## Builder Inversion

Each resource slice exports a `NewController` factory in its `backend/kubernetes/controller.go`. The factory assembles the full controller stack internally — the Kubernetes repo adapter, the plugin handler, and the `framework/backend/kubernetes/controller.GenericController` — and returns a `framework/backend/kubernetes/builder.Reconciler`.

The CSP `cmd/main.go` performs assembly:
```go
controllerSet := frameworkbuilder.NewControllerSet()
controllerSet.Add(bsk8s.NewController(mgr.GetClient(), dynClient, bsPlugin, opts...))
controllerSet.Add(netk8s.NewController(mgr.GetClient(), dynClient, netPlugin, opts...))
controllerSet.Add(wsk8s.NewController(mgr.GetClient(), dynClient, wsPlugin, opts...))
controllerSet.SetupWithManager(mgr)
```

No framework package ever names a concrete resource type. The `framework/backend/kubernetes/builder.ControllerSet` is a generic `[]Reconciler` aggregator with no resource knowledge.

## Available Plugins

### Dummy Plugin (`csp/dummy/`)

The reference implementation. It logs every operation without communicating with any real backend. Use it to:

- Understand the plugin interface contract.
- Run integration tests locally without CSP credentials.
- Test the gateway and controller layers in isolation.

```bash
# Build the dummy plugin image
make -C csp/dummy build

# Start a local KIND cluster with the dummy plugin deployed
make -C csp/dummy kind-start

# Run integration tests
make -C csp/dummy test-integration

# Tear down
make -C csp/dummy kind-stop
```

### IONOS Plugin (`csp/ionos/`)

Provisions IONOS Cloud resources using [Crossplane](https://crossplane.io/) with the `provider-upjet-ionoscloud` provider. The plugin introduces its own internal controller layer to bridge the ECP resource model and the Crossplane managed resource model.

**Updates are not implemented.** Every `Update` is a no-op, so an edit to a live IONOS-backed resource is accepted and stored but never reaches the provider. It is deliberately *not* reported with `ErrNotSupported`: `Update` is level-triggered and the plugin has no observed state to diff against, so it cannot tell a resource nobody touched from one carrying a change it must refuse — returning `ErrNotSupported` would stamp `UpdateFailed` on every healthy IONOS-backed resource and leave the condition meaning nothing.

**Prerequisites:**
- Kubernetes cluster with Crossplane installed
- IONOS API token

**Deployment:**
```bash
# Install Crossplane + IONOS provider (requires Helm)
make -C csp/ionos install-all

# Or install on an existing regional cluster
make -C csp/ionos install-on-regional
```

See `csp/ionos/README.md` for full deployment instructions, including token secret setup and provider configuration.

**Conformance** — the conformance suite (`secatest`) is plugin-generic and lives in
the test harness. Each plugin ships its own single-plugin delegator image
(`delegator-<plugin>`, built from `csp/<plugin>/cmd`); `CONFORMANCE_PLUGIN` on the
`conformance-deploy` target selects which one to build and deploy (the chart picks
the matching image and RBAC from its `plugin` value). Only `dummy` is
self-contained; like aruba, `ionos` only reconciles once its backend (Crossplane +
the IONOS provider, see `csp/ionos/deploy`) is installed in the cluster, so those
two run as a two-phase flow:
```bash
# dummy plugin — self-contained one-shot on KIND
make -C test kind-conformance

# aruba / ionos — deploy the stack with the plugin, provision its backend, then run
make -C test conformance-deploy CONFORMANCE_PLUGIN=aruba   # or CONFORMANCE_PLUGIN=ionos
#   ... install the plugin's backend (aruba operator / Crossplane + IONOS provider) ...
make -C test conformance
```

The IONOS plugin additionally keeps a standalone secatest flow for the full,
realistic split global/regional demo (separate clusters, Crossplane provider,
NodePort), in `test/conformance/ionos/` and delegated from the single Makefile:
```bash
make -C test conformance-ionos-scaffolding   # build images, create demo clusters, install plugin
make -C test conformance-ionos               # run secatest via NodePort
make -C test conformance-ionos-clean         # tear down
```

### Aruba Plugin (`csp/aruba/`)

Direct CSP adapter for Aruba Cloud, without a Crossplane layer.

## Test Harness (`test/`)

The `test/` module tests the full ECP stack on KIND. It has four kinds of
suite, all driven from one `Makefile`. Components are auto-discovered from the
`internal/build/` directory.

- **integration** (`test/integration/`) — each component in isolation. Every
  `kind-deploy-<component>` also deploys the fixtures its suite needs, so it is a
  complete setup for the matching `kind-integration-<component>`:
  - **`gateway-regional`** / **`gateway-global`** test only REST↔CR translation
    (asserting HTTP responses — never reconciled status). Each needs only its own
    gateway plus `test-data`.
  - **`delegator`** tests reconciliation: the dummy-plugin controllers drive CRs to
    `Active`. It needs `test-data`.
- **e2e** (`test/e2e/`) — the whole stack in one run: it drives the SECA API and
  asserts resources reconcile down to the delegator plugin. Single cluster.
- **multicluster e2e** (`test/e2e/multicluster/`) — the split topology: global gateway
  in one cluster, regional gateway + delegator in another, joined only by the Region CR
  the global gateway advertises. The suite gets only the global endpoint and must
  discover the regional API from the region catalog. Needs its own cluster pair, so it
  is not part of `test-all`.
- **conformance** — runs `secatest` against the stack, generic across plugins.

```bash
# Integration: deploy a suite's components and run it
make -C test kind-start
make -C test kind-deploy-gateway-regional
make -C test kind-integration-gateway-regional

# One shot (build, load, deploy the dummy stack, run the suite)
make -C test kind-e2e            # e2e only
make -C test kind-integration    # every integration suite
make -C test kind-test-all       # both

# Split global/regional topology (its own cluster pair)
make -C test kind-multicluster-e2e
make -C test kind-multicluster-stop

# Tear down
make -C test kind-stop
```

All suites run against the same stack, deployed with one authenticator
(`AUTH_PLUGIN=dummy|jwt`, default `dummy`) — `make -C test kind-test-all AUTH_PLUGIN=jwt`
runs everything against signed JWTs instead.

The stack is deployed from the shipped Helm charts (`charts/ecp/` for the gateways,
`charts/delegator/` for the delegator, whose `plugin` value is the same one selected here),
so a test run exercises the charts a real install uses. See `test/README.md` for the full
workflow. The test module (`test`) is excluded from the standard per-module CI checks (see
the exclude list in `ci/scripts/go-modules.sh`).

## Writing a New Plugin

1. **Create the module:**
   ```bash
   mkdir -p csp/<name>
   cd csp/<name>
   go mod init github.com/eu-sovereign-cloud/ecp/csp/<name>
   ```

2. **Add `require` and `replace` directives** for `framework` and `resource`:
   ```
   require (
       github.com/eu-sovereign-cloud/ecp/framework v0.0.1
       github.com/eu-sovereign-cloud/ecp/resource  v0.0.1
   )
   replace (
       github.com/eu-sovereign-cloud/ecp/framework => ../../framework
       github.com/eu-sovereign-cloud/ecp/resource  => ../../resource
   )
   ```

3. **Register in the workspace:**
   ```bash
   make workspace-use-add RELPATH=csp/<name>
   make workspace-sync
   ```

4. **Implement the plugin interfaces** from each resource slice's `backend/kubernetes/plugin.go`. Use `csp/dummy/` as a reference — it is the simplest complete implementation. `Update` is the one with a contract worth reading first (see above): it is level-triggered, so it must be idempotent and must not write when nothing has drifted. A plugin that cannot apply a given change should say so with `backend.ErrNotSupported` rather than returning `nil`, which would claim the change had been applied when nothing happened.

5. **Wire controllers in `cmd/main.go`** using builder inversion: instantiate each plugin, call each slice's `NewController`, add to `frameworkbuilder.NewControllerSet()`, then call `SetupWithManager(mgr)`.

6. **Add a Makefile** following the dummy plugin pattern with at minimum: `build`, `deploy`, `kind-start`, `kind-stop`.

7. **Commit** `go.work` and `go.work.sum`. CI auto-discovers the new module via `print-paths-filter`.
