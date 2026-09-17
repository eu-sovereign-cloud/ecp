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

Each plugin covers a different set of verticals — aruba reconciles workspaces,
storage (block storage and images) network and compute, dummy reconciles every
vertical, ionos covers compute, storage and network. The RBAC follows the
plugin, so switching `plugin` on an existing release re-grants the role to
match.

## Prerequisites

- The ECP CRDs, installed by the [`ecp`](../ecp) chart (or
  `kubectl apply -f charts/ecp/crds/`).
- Install into the **same cluster as the regional gateway** — the delegator
  reconciles the CRs that gateway writes.
- The backend for your plugin, installed **out of band**. Until it is present,
  resources stay pending. (For aruba, `test/conformance/aruba` installs it for
  you — see `test/conformance/aruba/README.md`.)

## Installing

Each plugin ships its own image, published as
`ghcr.io/eu-sovereign-cloud/ecp/delegator-<plugin>` on every `v*` tag (see
`.github/workflows/image-release.yaml`), and `image.tag` defaults to the
chart's appVersion — so the defaults resolve with no override. `dummy` is the
exception: it is not published, so it needs `image.repository` pointed at a
locally built, side-loaded image.

```bash
helm install ecp-delegator charts/delegator \
  --namespace ecp --create-namespace \
  --set plugin=aruba
```

The [`ecp`](../ecp) chart can also pull this one in as a subchart
(`--set ecp-delegator.enabled=true --set ecp-delegator.plugin=aruba`), which
ties both to one release and one version. Install standalone, as above, to
version and upgrade the delegator independently of the gateways.

## Values

See [values.yaml](values.yaml) for the full commented list. The notable ones:

| Key | Default | Notes |
|-----|---------|-------|
| `plugin` | `""` | **Required** — `aruba`, `dummy` or `ionos`; also selects the RBAC granted |
| `image.repository` | `""` → `ghcr.io/eu-sovereign-cloud/ecp/delegator-<plugin>` | Override to mirror the image into your own registry, or for `plugin=dummy`, which is not published |
| `replicaCount` | `1` | Keep at 1: the delegator runs without leader election |
| `rbac.create` | `true` | ClusterRole scoped to the selected plugin's controller set |

`helm lint`/CI note: because `plugin` has no default, lint with the CI values:
`helm lint charts/delegator -f charts/delegator/ci/default-values.yaml`.
