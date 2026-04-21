# Plugin System

This document describes the ECP plugin architecture, the available CSP plugins, and how to implement a new one.

## Overview

CSP plugins are the outermost layer of the delegator domain. They implement a set of interfaces defined in `foundation/delegator/pkg/plugin/` and are called by the Kubernetes controllers to perform the actual provisioning and management of cloud resources.

Each plugin is a separate Go module under `foundation/plugin/`, keeping CSP-specific dependencies isolated from the rest of the codebase.

## Plugin Interface

The plugin interface is defined in `foundation/delegator/pkg/plugin/`:

**`Workspace` interface** (`workspace.go`):
```go
type Workspace interface {
    Create(ctx context.Context, resource *regional.WorkspaceDomain) error
    Delete(ctx context.Context, resource *regional.WorkspaceDomain) error
}
```

**`BlockStorage` interface** (`block_storage.go`):
```go
type BlockStorage interface {
    Create(ctx context.Context, resource *regional.BlockStorageDomain) error
    Delete(ctx context.Context, resource *regional.BlockStorageDomain) error
    IncreaseSize(ctx context.Context, resource *regional.BlockStorageDomain) error
}
```

A plugin implements these interfaces for each resource type it supports. The delegator controllers call the appropriate interface method when a reconciliation event requires provisioning or deprovisioning.

## Available Plugins

### Dummy Plugin (`foundation/plugin/dummy/`)

The reference implementation. It logs every operation without communicating with any real backend. Use it to:

- Understand the plugin interface contract.
- Run integration tests locally without CSP credentials.
- Test the delegator and gateway layers in isolation.

```bash
# Build the dummy plugin image
make -C foundation/plugin/dummy build

# Start a local KIND cluster with the dummy plugin deployed
make -C foundation/plugin/dummy kind-start

# Run integration tests
make -C foundation/plugin/dummy test-integration

# Tear down
make -C foundation/plugin/dummy kind-stop
```

### IONOS Plugin (`foundation/plugin/ionos/`)

Provisions IONOS Cloud resources using [Crossplane](https://crossplane.io/) with the `provider-upjet-ionoscloud` provider. The plugin introduces its own internal controller layer to bridge the ECP resource model and the Crossplane managed resource model.

**Prerequisites:**
- Kubernetes cluster with Crossplane installed
- IONOS API token

**Deployment:**
```bash
# Install Crossplane + IONOS provider (requires Helm)
make -C foundation/plugin/ionos/deploy install-all

# Or install on an existing regional cluster
make -C foundation/plugin/ionos/deploy install-on-regional
```

See `foundation/plugin/ionos/README.md` for full deployment instructions, including token secret setup and provider configuration.

**IONOS E2E tests** (`foundation/ionos_e2e/`):
```bash
# Full scaffolding + test in one shot
make -C foundation/ionos_e2e secatest-all

# Step by step
make -C foundation/ionos_e2e secatest-scaffolding
make -C foundation/ionos_e2e secatest
make -C foundation/ionos_e2e secatest-clean
```

### Aruba Plugin (`foundation/plugin/aruba/`)

Direct CSP adapter for Aruba Cloud, without a Crossplane layer.

## E2E Test Harness (`foundation/plugin/e2e/`)

A multi-component test harness that tests the full ECP stack (gateway + delegator + plugin) end-to-end on a KIND cluster. Components are auto-discovered from the `build/` directory.

```bash
# Start KIND cluster, load all images, deploy all components
make -C foundation/plugin/e2e kind-start

# Build all component images
make -C foundation/plugin/e2e build-all

# Run all tests
make -C foundation/plugin/e2e test-all

# Tear down
make -C foundation/plugin/e2e kind-stop
```

The e2e module (`foundation/plugin/e2e`) is excluded from the standard per-module CI checks (`GO_MODULES_EXCLUDE` in `.common.mk`).

## Writing a New Plugin

1. **Create the module:**
   ```bash
   mkdir -p foundation/plugin/<name>
   cd foundation/plugin/<name>
   go mod init github.com/eu-sovereign-cloud/ecp/foundation/plugin/<name>
   ```

2. **Register in the workspace:**
   ```bash
   make workspace-use-add RELPATH=foundation/plugin/<name>
   # Add replace directive if the plugin imports other workspace members:
   go work edit -replace github.com/eu-sovereign-cloud/ecp/foundation/plugin/<name>=./foundation/plugin/<name>
   make workspace-sync
   ```

3. **Implement the plugin interfaces** from `foundation/delegator/pkg/plugin/`. Use `foundation/plugin/dummy/` as a reference — it is the simplest complete implementation.

4. **Add a Makefile** following the dummy plugin pattern with at minimum: `build`, `deploy`, `kind-start`, `kind-stop`.

5. **Commit** `go.work` and `go.work.sum`. CI auto-discovers the new module via `print-paths-filter`.
