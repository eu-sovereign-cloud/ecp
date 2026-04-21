# ECP — European Control Plane

A Kubernetes-native distributed control plane for managing cloud resources across multiple cloud service providers (CSPs).

ECP exposes a unified, declarative REST API for provisioning and managing cloud resources. All state is persisted as Kubernetes Custom Resources, enabling compatibility with existing Kubernetes tooling and GitOps workflows. See [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md) for the full design.

## Repository Layout

```
foundation/
├── persistence/      # CRD definitions and generated API types
├── gateway/          # Global and regional REST API servers
├── delegator/        # Kubernetes controllers
└── plugin/
    ├── dummy/        # Reference plugin implementation
    ├── ionos/        # IONOS CSP plugin (Crossplane-based)
    ├── aruba/        # Aruba CSP plugin
    └── e2e/          # End-to-end test harness
ci/
├── container/        # Dockerfile layers: builder, tools, dev, runner
├── scripts/          # CI and dev automation scripts
└── tools/            # Pinned Go dev tool dependencies
modules/
└── go-sdk/           # Git submodule: shared OpenAPI specs and client SDK
doc/                  # Documentation
```

## Go Workspace

This is a Go monorepo managed with `go.work`. The workspace contains 8 modules:

| Module | Description |
|--------|-------------|
| `foundation/persistence` | CRD definitions, generated API types, K8s repository interfaces |
| `foundation/gateway` | Global and regional REST API servers |
| `foundation/delegator` | Kubernetes controllers and plugin interface |
| `foundation/plugin/dummy` | Reference plugin (no real backend) |
| `foundation/plugin/ionos` | IONOS CSP adapter via Crossplane |
| `foundation/plugin/aruba` | Aruba CSP adapter |
| `foundation/plugin/e2e` | End-to-end test harness |
| `ci/tools/go` | Pinned versions of Go development tools |

## Quick Start

**Prerequisites:** Docker (or Podman), `kubectl`, KIND.

> Go is **not** required on the host. All compilation runs inside the `builder` container image, which is pulled automatically on first use.

```bash
# Generate CRDs and typed Go models from OpenAPI specs
make generate-api

# Create local KIND clusters (global + regional)
make -C foundation/gateway create-dev-clusters

# Run the API servers (in separate terminals)
make -C foundation/gateway run-global-server
make -C foundation/gateway run-regional-server

# Run all tests
make test

# Lint all modules
make lint

# Full local validation gate (mirrors CI)
make pre-commit
```

For containerized development, persistent dev containers, and the full Makefile reference, see [doc/CI_DEVEX.md](doc/CI_DEVEX.md).

## Documentation

| Document | Description |
|----------|-------------|
| [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md) | System architecture, hexagonal design, resource model |
| [doc/CI_DEVEX.md](doc/CI_DEVEX.md) | Developer environment setup, Makefile targets, CI pipeline |
| [doc/CODEGEN.md](doc/CODEGEN.md) | Code generation pipeline (OpenAPI types, CRDs, controller-gen) |
| [doc/PLUGINS.md](doc/PLUGINS.md) | Plugin system: available plugins, interface, writing a new CSP plugin |
| [doc/CONTRIBUTING.md](doc/CONTRIBUTING.md) | Contribution guidelines, PR conventions, branch model |

## Current Version

`v0.1.0-alpha1-preview` — API surface and CRD schemas are subject to breaking changes before v1.0.

---

## Funding

This open-source project is sponsored by **Aruba & IONOS SE** and has received public funding from the European Union NextGenerationEU within the IPCEI-CIS program.

![SECA Funding Logo](https://github.com/eu-sovereign-cloud/.github/raw/main/profile/SECA-Funding-Logo.png)
