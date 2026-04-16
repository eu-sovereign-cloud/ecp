# ECP — European Control Plane

A Kubernetes-native distributed control plane for managing cloud resources across multiple cloud service providers (CSPs).

## Overview

ECP provides a unified, declarative REST API for provisioning and managing cloud resources. All resources are persisted as Kubernetes Custom Resources (CRs), enabling compatibility with existing Kubernetes tooling and GitOps workflows.

The system has three main layers:

1. **Gateway** — REST API servers (global and regional) that expose resource endpoints. Generated from the same OpenAPI specs as the client SDK ([go-sdk](https://github.com/eu-sovereign-cloud/go-sdk)), ensuring no encoding gap between client and server.
2. **Delegator** — Kubernetes controllers that watch CRs, validate state transitions, and delegate provisioning to CSP plugins.
3. **Plugins** — CSP-specific adapters that perform the actual resource provisioning (e.g., IONOS, Aruba).

See [doc/README.md](doc/README.md) for full architecture documentation.

## Repository Structure

```
foundation/
├── api/          # CRD definitions and generated API types
├── gateway/      # Global and regional REST API servers
├── delegator/    # Kubernetes controllers
└── plugin/
    ├── dummy/    # Reference plugin implementation
    ├── ionos/    # IONOS CSP plugin
    └── aruba/    # Aruba CSP plugin
```

## Prerequisites

- Docker (or Podman)
- `kubectl`
- KIND (for local development)

> Go is **not** required on the host. All compilation runs inside the `builder`
> container image, which is pulled automatically on first use (see below).

## Builder image

The `builder` image contains the Go toolchain and all codegen/lint tools.
It is published to `ghcr.io/eu-sovereign-cloud/ecp-builder` by CI and pulled
automatically when you run any Makefile target that needs it:

```bash
make tools-build    # pulls the pinned builder, then builds the tools image
make dev-build      # same — no manual pull needed
```

The pinned digest is stored in `.builder-digest` (committed to git) and
updated by an automated CI pull-request whenever the builder inputs change.

**To modify the builder itself** (i.e., editing `ci/container/builder/`):

```bash
make builder-build BUILDER_SOURCE=local   # rebuild from local Dockerfile
make tools-build   BUILDER_SOURCE=local   # use the local build downstream
```

> **First-time setup for the registry** (maintainers only, one-time):
> After the first `builder-publish.yaml` CI run succeeds, go to
> `https://github.com/orgs/eu-sovereign-cloud/packages/container/ecp-builder/settings`
> and set the package visibility to **Public** so anonymous pulls work.

## Getting Started

### Generate code and CRDs

```bash
make generate-api
```

### Set up local development clusters

```bash
make -C foundation/gateway create-dev-clusters
```

### Run the API servers

```bash
# Global API server
make -C foundation/gateway run-global-server

# Regional API server (in a separate terminal)
make -C foundation/gateway run-regional-server
```

### Run tests

```bash
go test -race ./...

# Integration tests (dummy plugin, requires KIND)
make -C foundation/plugin/dummy test-integration
```

### Lint

```bash
golangci-lint run --config .golangci.yml
```

## Resource Model

**Global resources:**
- `Region` — available regions (read-only)

**Regional resources:**
- `Workspace` — logical grouping within a tenant; creation triggers a dedicated namespace
- `BlockStorage` — block storage volumes
- `Network` — network resources
- `StorageSKU` / `NetworkSKU` — available SKU options (read-only)

Deleting a Tenant cascades to all its Workspaces; deleting a Workspace cascades to all resources within it.

## API Endpoints

| Server   | Default port | Example endpoints                          |
|----------|--------------|--------------------------------------------|
| Global   | 8080         | `GET /regions`, `GET /regions/{id}`        |
| Regional | 8080         | `GET/POST /workspaces`, `GET/POST /block-storages`, `GET /skus` |

## CI

Pull request checks are defined in [`.github/workflows/pre-merge.yaml`](.github/workflows/pre-merge.yaml) and include module-aware testing, linting (`golangci-lint`), and security scanning (`govulncheck`, `gosec`).

---

## 💰 Funding

This open-source project is sponsored by **Aruba & IONOS SE** and has received public funding from the European Union NextGenerationEU within the IPCEI-CIS program.

![SECA Funding Logo](https://github.com/eu-sovereign-cloud/.github/raw/main/profile/SECA-Funding-Logo.png)
