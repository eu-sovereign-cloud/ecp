# ECP — European Control Plane 

A Kubernetes-native distributed control plane for managing cloud resources across multiple cloud service providers (CSPs).

ECP exposes a unified, declarative REST API for provisioning and managing cloud resources. All state is persisted as Kubernetes Custom Resources, enabling compatibility with existing Kubernetes tooling and GitOps workflows. See [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md) for the full design.

## Install with Helm

Released charts are published to the repository at
<https://eu-sovereign-cloud.github.io/ecp/>:

```bash
helm repo add ecp https://eu-sovereign-cloud.github.io/ecp/
helm repo update
helm search repo ecp --devel
```

| Chart | Contains |
|-------|----------|
| `ecp/ecp` | Global + regional gateways, and the ECP CRDs |
| `ecp/ecp-delegator` | The delegator — reconciles the CRs through **one** CSP plugin. Also available as a subchart of `ecp/ecp` |

> Releases so far are **prereleases** (`0.0.2-alpha`), which Helm skips unless
> you ask for them: pass `--version 0.0.2-alpha` (as below) or `--devel`.

### The common setup: gateways + delegator, one plugin

Both gateways and the delegator in one cluster, with the delegator pulled in as
a subchart so one release covers the whole stack. `ecp-delegator.plugin` is what
picks the CSP — it selects both the delegator image and the RBAC the chart grants:

```bash
# Aruba
helm install ecp ecp/ecp --version 0.0.2-alpha \
  --namespace ecp --create-namespace \
  --set gatewayRegional.region=itbg-bergamo \
  --set ecp-delegator.enabled=true \
  --set ecp-delegator.plugin=aruba

# IONOS — same install, different plugin
helm install ecp ecp/ecp --version 0.0.2-alpha \
  --namespace ecp --create-namespace \
  --set gatewayRegional.region=<your-region> \
  --set ecp-delegator.enabled=true \
  --set ecp-delegator.plugin=ionos
```

`gatewayRegional.region` is **required** and must be the region this gateway
serves. Install `ecp/ecp-delegator` standalone instead of enabling the subchart
if you want to version and upgrade it independently of the gateways.

Each plugin reconciles into a backend you install **out of band** — until it is
there, resources are accepted and stay pending:

| `plugin` | Reconciles by | Backend to install first |
|----------|---------------|--------------------------|
| `aruba` | writing `arubacloud.com` CRs | [arubacloud-resource-operator](https://github.com/Arubacloud/arubacloud-resource-operator) + Aruba credentials |
| `ionos` | writing Crossplane managed resources | Crossplane + `provider-upjet-ionoscloud` + an IONOS token ([`csp/ionos/deploy`](csp/ionos/deploy)) |
| `dummy` | nothing — marks resources Active in-process | none; development only, image not published |

For the split topology (global gateway in one cluster, regional gateway +
delegator in each region), install the chart once per cluster with
`--set gatewayRegional.enabled=false` on the global one and
`--set gatewayGlobal.enabled=false` on each regional one.

Helm installs the CRDs on first install but never upgrades them; on an upgrade
apply them yourself with `kubectl apply -f` from the chart's `crds/`.

### Enabling JWT authentication

**Auth is off by default** — the API is unauthenticated in that mode, so do not
expose it outside the cluster until you turn it on. Turning it on enables bearer
token authentication *and* SECA RBAC on **both** gateways.

The `jwt` plugin verifies standard signed JWTs against a key you supply, with
the token's `alg` header pinned to `auth.jwt.signingMethod` (`ES256` by default)
— the pin is what defeats algorithm confusion. Generate an ES256 pair and hand
the gateways only the public half:

```bash
openssl ecparam -name prime256v1 -genkey -noout -out jwt-key.pem   # private — your issuer signs with it
openssl ec -in jwt-key.pem -pubout -out jwt-key.pub                # public  — the gateways verify with it

helm install ecp ecp/ecp --version 0.0.2-alpha \
  --namespace ecp --create-namespace \
  --set gatewayRegional.region=itbg-bergamo \
  --set ecp-delegator.enabled=true \
  --set ecp-delegator.plugin=aruba \
  --set auth.enabled=true \
  --set auth.plugin=jwt \
  --set-file auth.jwt.key=jwt-key.pub
```

Callers then send the JWT verbatim: `Authorization: Bearer <header>.<payload>.<signature>`.
It must carry `sub` (the subject RBAC is evaluated for) and `exp` (a
never-expiring token is rejected), plus an optional `scope` object to narrow it
further. Roles are **never** read from the token — entitlements come only from
the `Role` / `RoleAssignment` CRs in the tenant namespace.

Other knobs: `auth.jwt.existingSecret` to reference a Secret you already manage
(key `jwt.pub`) instead of `--set-file`; `auth.authz.enabled=false` for
authn-only. The `dummy` plugin (`auth.plugin=dummy`, the default) takes a
`username → password` map in `auth.dummyUsers.users` and verifies no signature
at all — development and testing only.

See [doc/AUTH.md](doc/AUTH.md) for the token formats, down-scoping and the RBAC
algorithm, and [charts/ecp/README.md](charts/ecp/README.md) for every value.

## Repository Layout

```
framework/            # Resource-agnostic SDK (horizontal axis)
├── kernel/           #   All abstractions: ports, Scope, Error, validation
│   └── port/         #     authn.Authenticator, authz.Checker — auth port interfaces
├── backend/          #   Kubernetes backend: adapter, schema/v1 CRDs, codegen, controller, builder
│   └── kubernetes/   #     adapter, labels, convert, schema/v1, controller, builder, cmd
└── frontend/         #   HTTP server, kubeclient, logger, config
    └── middleware/   #     NewAuthentication, NewAuthorization, SECAClaimExtractor, Chain
resource/             # Data vocabulary + per-resource slices (vertical axis)
├── common/           #   Shared domain, frontend, backend helpers
└── <group>/vN/<resource>/
    ├── domain.go     #   Canonical type + identity consts (package <resource>)
    ├── frontend/rest/#   REST↔domain converters + HTTP handlers (per-group, shared handler)
    └── backend/kubernetes/ # CR types, adapters, controller, plugin interface + handler
gateway/              # Global and regional REST API server binary
├── internal/authn/   #   DummyAuthenticator (bearer-token dev/test auth)
├── internal/authz/   #   seca/ — SECA RBAC Checker + CachedChecker
└── internal/auth/    #   Build, ProviderMWs, StartChecker — opt-in wiring
csp/
├── dummy/            # Reference plugin (no real backend)
├── ionos/            # IONOS CSP plugin (Crossplane-based)
└── aruba/            # Aruba CSP plugin
test/                 # Test harness: integration, e2e and conformance suites
├── integration/      # Per-component suites (delegator, gateway-global/-regional)
├── e2e/              # End-to-end suites: single-cluster, plus multicluster/ (split topology)
├── conformance/      # secatest conformance harnesses (ionos, aruba)
└── internal/         # Shared infra: build, deploy, scripts, cmd, testenv, authhelper, context
ci/
├── container/        # Dockerfile layers: builder, tools, dev, runner
├── scripts/          # CI and dev automation scripts
└── tools/            # Pinned Go dev tool dependencies
charts/               # Helm charts
├── ecp/              # The global and regional gateways
│   ├── crds/         # Generated Kubernetes CRD YAML (all 18 resource slices)
│   └── templates/    # Gateway Deployments, Services, RBAC, ingress
└── delegator/        # The delegator, one plugin set per install
modules/
└── go-sdk/           # Git submodule: shared OpenAPI specs and client SDK
doc/                  # Documentation
```

## Go Workspace

This is a Go monorepo managed with `go.work`. The workspace contains 8 first-party modules:

| Module | Path | Description |
|--------|------|-------------|
| `framework` | `./framework` | Resource-agnostic SDK (kernel, backend, frontend) |
| `resource` | `./resource` | Domain vocabulary + all resource slices |
| `gateway` | `./gateway` | Global and regional REST API server binary |
| `csp/dummy` | `./csp/dummy` | Reference plugin (no real backend) |
| `csp/ionos` | `./csp/ionos` | IONOS CSP adapter via Crossplane |
| `csp/aruba` | `./csp/aruba` | Aruba CSP adapter |
| `test` | `./test` | Test harness (integration, e2e, conformance) |
| `ci/tools/go` | `./ci/tools/go` | Pinned versions of Go development tools |

**Module boundary**: `framework ↛ resource` is compiler-enforced. `resource` and `gateway` depend on `framework`. CSP plugins depend on both `framework` and `resource`. See [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md).

## Quick Start

**Prerequisites:** Docker (or Podman), `kubectl`, KIND.

> Go is **not** required on the host. All compilation runs inside the `builder` container image, which is pulled automatically on first use.

```bash
# If you have a frash repository clone, you need to get the sub modules:

    make submodules

If this failes because of SSH access, one way to solve it is

    git config --global url."https://github.com/".insteadOf "git@github.com:"

Then run `make modules again`.


# Generate CRDs and typed Go models from OpenAPI specs
make generate-api

# Start a local KIND cluster with the reference plugin (includes global + regional)
make -C csp/dummy kind-start

# Run the API servers (in separate terminals)
go run ./gateway globalapiserver
go run ./gateway regionalapiserver --region local -p 8081

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
| [doc/AUTH.md](doc/AUTH.md) | Authentication & authorization — bearer-token format, token down-scoping, SECA RBAC algorithm, config flags |
| [doc/CI_DEVEX.md](doc/CI_DEVEX.md) | Developer environment setup, Makefile targets, CI pipeline |
| [doc/CODEGEN.md](doc/CODEGEN.md) | Code generation pipeline (OpenAPI types, CRDs, controller-gen) |
| [doc/PLUGINS.md](doc/PLUGINS.md) | Plugin system: interface, builder inversion, writing a new CSP plugin |
| [doc/CONTRIBUTING.md](doc/CONTRIBUTING.md) | Contribution guidelines, import alias convention, PR conventions |
| [doc/CONVENTIONS.md](doc/CONVENTIONS.md) | Go style conventions — naming, initialisms, conversion functions, structural symmetry |
| [doc/AUTH-SPEC-REVIEW.md](doc/AUTH-SPEC-REVIEW.md) | Auth findings — token model and SECA spec alignment review (record) |
| [charts/ecp/README.md](charts/ecp/README.md) | Gateway chart — topology toggles, auth values, CRDs, full value list |
| [charts/delegator/README.md](charts/delegator/README.md) | Delegator chart — plugin selection, per-plugin RBAC and backends |

## Current Version

`v0.0.2-alpha` — the latest release, and the version the charts and images above
are published under. API surface and CRD schemas are subject to breaking changes
before v1.0.

---

## Funding

This open-source project is sponsored by **Aruba & IONOS SE** and has received public funding from the European Union NextGenerationEU within the IPCEI-CIS program.

![SECA Funding Logo](https://github.com/eu-sovereign-cloud/.github/raw/main/profile/SECA-Funding-Logo.png)
