# ecp-delegator

Helm chart for the ECP **delegator** — the controller that watches the ECP
Custom Resources (written by the regional gateway) and reconciles them through
one CSP plugin.

One image serves every CSP, so one chart does too: the required `plugin` value
selects the controller set the delegator loads **and** the RBAC the chart
grants it. There is no default — an install must say which cloud it is for.

| `plugin` | Reconciles by | Backend you must install |
|----------|---------------|--------------------------|
| `aruba` | writing `arubacloud.com` CRs | [arubacloud-resource-operator](https://github.com/Arubacloud/arubacloud-resource-operator) + Aruba credentials |
| `ionos` | writing crossplane managed resources | Crossplane + the IONOS provider (`csp/ionos/deploy`) |
| `dummy` | nothing — in-process simulation | none |

> ⚠️ `plugin=dummy` marks resources Active without provisioning anything. It
> exists for development and testing; never run it in production.

Each plugin covers a different set of verticals — aruba reconciles workspaces
and block storage, dummy reconciles every vertical, ionos covers compute,
storage and network. The RBAC follows the plugin, so switching `plugin` on an
existing release re-grants the role to match.

## Prerequisites

- The ECP CRDs, installed by the [`ecp`](../chart) chart (or
  `kubectl apply -f chart/crds/`).
- Install into the **same cluster as the regional gateway** — the delegator
  reconciles the CRs that gateway writes.
- The backend for your plugin, installed **out of band**. Until it is present,
  resources stay pending. (For aruba there is no in-repo tooling yet — see
  `test/conformance/aruba/README.md`.)

## Installing

The delegator image is published to `ghcr.io/eu-sovereign-cloud/ecp/delegator`
on every `v*` tag (see `.github/workflows/image-release.yaml`), and
`image.tag` defaults to the chart's appVersion — so the defaults resolve with
no override.

```bash
helm install ecp-delegator chart-delegator \
  --namespace ecp --create-namespace \
  --set plugin=aruba
```

## Values

See [values.yaml](values.yaml) for the full commented list. The notable ones:

| Key | Default | Notes |
|-----|---------|-------|
| `plugin` | `""` | **Required** — `aruba`, `dummy` or `ionos`; also selects the RBAC granted |
| `image.repository` | `ghcr.io/eu-sovereign-cloud/ecp/delegator` | Override only to mirror the image into your own registry |
| `replicaCount` | `1` | Keep at 1: the delegator runs without leader election |
| `rbac.create` | `true` | ClusterRole scoped to the selected plugin's controller set |

`helm lint`/CI note: because `plugin` has no default, lint with the CI values:
`helm lint chart-delegator -f chart-delegator/ci/default-values.yaml`.
