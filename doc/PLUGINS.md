# Plugin System

This document describes the ECP plugin architecture, the available CSP plugins, and how to implement a new one.

## Overview

CSP plugins implement the resource interfaces defined in each resource slice's `backend/kubernetes/plugin.go` and are called by the per-slice Kubernetes controllers to perform the actual provisioning and management of cloud resources.

Each plugin is a separate Go module under `csp/`, keeping CSP-specific dependencies isolated from the rest of the codebase.

## Plugin Interfaces

Plugin interfaces live in each resource slice at `resource/{group}/vN/{resource}/backend/kubernetes/plugin.go`. This co-locates the interface with the controller and handler that use it — no framework package ever names a concrete resource.

**`WorkspacePlugin` interface** (`resource/workspace/v1/backend/kubernetes/plugin.go`):
```go
type WorkspacePlugin interface {
    Create(ctx context.Context, resource *wsdom.Workspace) error
    Delete(ctx context.Context, resource *wsdom.Workspace) error
}
```

**`BlockStoragePlugin` interface** (`resource/storage/v1/block-storage/backend/kubernetes/plugin.go`):
```go
type BlockStoragePlugin interface {
    Create(ctx context.Context, resource *bsdom.BlockStorage) error
    Delete(ctx context.Context, resource *bsdom.BlockStorage) error
    IncreaseSize(ctx context.Context, resource *bsdom.BlockStorage) error
}
```

**`NetworkPlugin` interface** (`resource/network/v1/network/backend/kubernetes/plugin.go`):
```go
type NetworkPlugin interface {
    Create(ctx context.Context, resource *netdom.Network) error
    Delete(ctx context.Context, resource *netdom.Network) error
}
```

A CSP plugin implements these interfaces for each resource type it supports.

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

**Prerequisites:**
- Kubernetes cluster with Crossplane installed
- IONOS API token

**Deployment:**
```bash
# Install Crossplane + IONOS provider (requires Helm)
make -C csp/ionos/deploy install-all

# Or install on an existing regional cluster
make -C csp/ionos/deploy install-on-regional
```

See `csp/ionos/README.md` for full deployment instructions, including token secret setup and provider configuration.

**Conformance** — the conformance suite (`secatest`) is plugin-generic and lives in
the test harness. The delegator can load any of the plugin sets (`dummy`, `aruba`,
`ionos`) via `PLUGIN`/`CONFORMANCE_PLUGIN`, so the same flow targets any of them —
though, like aruba, `ionos` only reconciles once its backend (Crossplane + the IONOS
provider, see `csp/ionos/deploy`) is installed in the cluster:
```bash
make -C test kind-conformance                          # dummy plugin
make -C test conformance CONFORMANCE_PLUGIN=aruba       # aruba plugin
make -C test conformance CONFORMANCE_PLUGIN=ionos       # ionos plugin (needs its backend)
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

The `test/` module tests the full ECP stack on a KIND cluster. It has three kinds of
suite, all driven from one `Makefile`. Components are auto-discovered from the
`build/` directory.

- **integration** (`test/integration/`) — each component in isolation. Every
  `kind-deploy-<component>` also deploys the fixtures its suite needs, so it is a
  complete setup for the matching `kind-test-<component>`:
  - **`gateway-regional`** / **`gateway-global`** test only REST↔CR translation
    (asserting HTTP responses — never reconciled status). Each needs only its own
    gateway plus `test-data`.
  - **`delegator`** tests reconciliation: the dummy-plugin controllers drive CRs to
    `Active`. It needs `test-data`.
- **e2e** (`test/e2e/`) — the whole stack in one run: it drives the SECA API and
  asserts resources reconcile down to the delegator plugin.
- **conformance** — runs `secatest` against the stack, generic across plugins.

```bash
# Integration: deploy a suite's components and run it
make -C test kind-start
make -C test kind-deploy-gateway-regional
make -C test kind-test-gateway-regional

# End-to-end (one shot: build, load, deploy the dummy stack, run the suite)
make -C test kind-e2e

# Tear down
make -C test kind-stop
```

See `test/README.md` for the full workflow. The test module (`test`) is excluded from
the standard per-module CI checks (see the exclude list in `ci/scripts/go-modules.sh`).

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

4. **Implement the plugin interfaces** from each resource slice's `backend/kubernetes/plugin.go`. Use `csp/dummy/` as a reference — it is the simplest complete implementation.

5. **Wire controllers in `cmd/main.go`** using builder inversion: instantiate each plugin, call each slice's `NewController`, add to `frameworkbuilder.NewControllerSet()`, then call `SetupWithManager(mgr)`.

6. **Add a Makefile** following the dummy plugin pattern with at minimum: `build`, `deploy`, `kind-start`, `kind-stop`.

7. **Commit** `go.work` and `go.work.sum`. CI auto-discovers the new module via `print-paths-filter`.
