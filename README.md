# ECP — European Control Plane

A Kubernetes-native distributed control plane for managing cloud resources across multiple cloud service providers (CSPs).

ECP exposes a unified, declarative REST API for provisioning and managing cloud resources. All state is persisted as Kubernetes Custom Resources, enabling compatibility with existing Kubernetes tooling and GitOps workflows. See [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md) for the full design.

## Repository Layout

```
framework/            # Resource-agnostic SDK (horizontal axis)
├── kernel/           #   All abstractions: ports, Scope, Error, validation
├── backend/          #   Kubernetes backend: adapter, schema/v1 CRDs, codegen, controller, builder
│   └── kubernetes/   #     adapter, labels, convert, schema/v1, controller, builder, cmd
└── frontend/         #   HTTP server, kubeclient, logger, config
resources/            # Data vocabulary + per-resource slices (vertical axis)
├── common/           #   Shared domain, frontend, backend helpers
└── <group>/<resource>/vN/
    ├── domain.go     #   Canonical type + identity consts (package v1)
    ├── frontend/rest/#   REST↔domain converters + HTTP handlers
    └── backend/kubernetes/ # CR types, adapters, controller, plugin interface + handler
gateway/              # Global and regional REST API server binary
csp/
├── dummy/            # Reference plugin (no real backend)
├── ionos/            # IONOS CSP plugin (Crossplane-based)
└── aruba/            # Aruba CSP plugin
test/
├── e2e/              # End-to-end test harness
└── ionos-e2e/        # IONOS-specific integration tests
ci/
├── container/        # Dockerfile layers: builder, tools, dev, runner
├── scripts/          # CI and dev automation scripts
└── tools/            # Pinned Go dev tool dependencies
modules/
└── go-sdk/           # Git submodule: shared OpenAPI specs and client SDK
doc/                  # Documentation
```

## Go Workspace

This is a Go monorepo managed with `go.work`. The workspace contains 8 first-party modules:

| Module | Path | Description |
|--------|------|-------------|
| `framework` | `./framework` | Resource-agnostic SDK (kernel, backend, frontend) |
| `resources` | `./resources` | Domain vocabulary + all resource slices |
| `gateway` | `./gateway` | Global and regional REST API server binary |
| `csp/dummy` | `./csp/dummy` | Reference plugin (no real backend) |
| `csp/ionos` | `./csp/ionos` | IONOS CSP adapter via Crossplane |
| `csp/aruba` | `./csp/aruba` | Aruba CSP adapter |
| `test/e2e` | `./test/e2e` | End-to-end test harness |
| `ci/tools/go` | `./ci/tools/go` | Pinned versions of Go development tools |

**Module boundary**: `framework ↛ resources` is compiler-enforced. `resources` and `gateway` depend on `framework`. CSP plugins depend on both `framework` and `resources`. See [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md).

## Quick Start

**Prerequisites:** Docker (or Podman), `kubectl`, KIND.

> Go is **not** required on the host. All compilation runs inside the `builder` container image, which is pulled automatically on first use.

```bash
# Generate CRDs and typed Go models from OpenAPI specs
make generate-api

# Start a local KIND cluster with the reference plugin (includes global + regional)
make -C csp/dummy kind-start

# Run the API servers (in separate terminals)
go run ./gateway globalapiserver
go run ./gateway regionalapiserver

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
| [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md) | DDD/hexagonal design, two-axis module topology, module DAG |
| [doc/CI_DEVEX.md](doc/CI_DEVEX.md) | Developer environment setup, Makefile targets, CI pipeline |
| [doc/CODEGEN.md](doc/CODEGEN.md) | Code generation pipeline (OpenAPI types, CRDs, controller-gen) |
| [doc/PLUGINS.md](doc/PLUGINS.md) | Plugin system: interface, builder inversion, writing a new CSP plugin |
| [doc/CONTRIBUTING.md](doc/CONTRIBUTING.md) | Contribution guidelines, import alias convention, PR conventions |

## Current Version

`v0.1.0-alpha1-preview` — API surface and CRD schemas are subject to breaking changes before v1.0.

---

## Funding

This open-source project is sponsored by **Aruba & IONOS SE** and has received public funding from the European Union NextGenerationEU within the IPCEI-CIS program.

![SECA Funding Logo](https://github.com/eu-sovereign-cloud/.github/raw/main/profile/SECA-Funding-Logo.png)
