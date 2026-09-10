# Aruba conformance (real backend)

Runs the SECA conformance suite (`secatest`) against the aruba plugin backed by a
**real Aruba account**, on a single KIND cluster, in one command:

```shell
make -C test conformance-aruba-all \
    ARUBA_CLIENT_ID=<client-id> \
    ARUBA_CLIENT_SECRET=<client-secret> \
    ARUBA_TENANT=<ARU-account>
```

That creates the cluster, installs the
[`arubacloud-resource-operator`](https://github.com/Arubacloud/helm-charts/tree/main/charts/arubacloud-resource-operator)
with those credentials, builds and side-loads the images, deploys the ECP stack
(both gateways + `delegator-aruba` + the `test-data` fixtures) and runs `secatest`
against the global gateway.

> ⚠️ **It provisions real, billable Aruba resources.** Tear down with
> `make -C test conformance-aruba-clean`, which deletes the SECA resources — and
> so, through the plugin and the operator, what they created in the CMP — *before*
> it deletes the cluster.

## Why one cluster (unlike IONOS)

[`../ionos/`](../ionos/) stands up a global and a regional cluster because its demo
is about the split control-plane topology. Aruba's backend is an **operator in the
same cluster**, not a second control plane, so the split adds nothing this run
exercises — and the multicluster topology already has [its own suite](../../README.md#multicluster-e2e-two-clusters).
The IONOS harness also deploys the gateways from hand-written manifests; this one
reuses the harness's own chart-based deploy path, so it only has to sequence
existing targets and install the operator.

## Prerequisites

Docker, KIND, kubectl, Helm, kustomize, make — the harness's usual set — plus:

- **An Aruba API client** (`clientId` / `clientSecret`), which is what the operator
  authenticates to the Aruba CMP with.
- **A real Aruba account** in `ARUBA_TENANT` (e.g. `ARU-348095`). It is both the
  SECA tenant the stack is seeded for and the tenant `secatest` drives, so the
  fixture SKUs land in the same `hex(sha3-224(tenant))` namespace the plugin reads
  them from. Against anything else nothing reaches `Active`.

## Targets

Reachable either from the harness root (`make -C test conformance-aruba-*`) or
directly in this directory (`make -C test/conformance/aruba *`):

| From `test/` | Here | Does |
|--------------|------|------|
| `conformance-aruba-all` | `conformance-all` | Everything: scaffolding, then the run. |
| `conformance-aruba-scaffolding` | `scaffolding` | Cluster + operator + images + stack. |
| `conformance-aruba` | `conformance` | Just `secatest`, against an already-scaffolded cluster. |
| `conformance-aruba-clean` | `clean` | Delete the provisioned resources, then the cluster. |
| — | `install-operator` | Just the operator (also re-runnable to rotate credentials). |

Iterating on a failure is therefore `conformance-aruba-scaffolding` once, then
`conformance-aruba` as often as you like.

## Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ARUBA_CLIENT_ID` | _(required)_ | Aruba API client id the operator authenticates with. |
| `ARUBA_CLIENT_SECRET` | _(required)_ | Its secret. |
| `ARUBA_TENANT` | _(required)_ | Real Aruba account; the SECA tenant everything is seeded and driven for. |
| `KIND_CLUSTER` | `aruba-conformance` | Cluster to create/use. Its own, so it never collides with `e2e-cluster`. |
| `OPERATOR_NAMESPACE` | `aruba-system` | Namespace the operator is installed into. |
| `OPERATOR_VERSION` | `1.0.1` | Chart version from `https://arubacloud.github.io/helm-charts`. |
| `CONFORMANCE_SCENARIOS` | `.*` | `secatest` scenario filter, e.g. `Storage.V1.BlockStorageLifeCycle`. |
| `CONFORMANCE_REGION` | `itbg-bergamo` | Must name a Region CR in [`test-data/regions.yaml`](../../internal/deploy/test-data/regions.yaml). |
| `CONFORMANCE_AUTH_TOKEN` | admin dummy token | The stack deploys with auth **on**, so the runner presents the `admin` fixture user, whose `ra-admin` assignment is unscoped. |
| `CONFORMANCE_RETRY_*` | `30` / `20` / `30` | Retry delay / interval / attempts — much longer than the dummy defaults, since every wait is a real CMP round-trip. |

Everything else comes from the shared runner,
[`internal/scripts/conformance.sh`](../../internal/scripts/conformance.sh) — it is
plugin-generic, and only the tenant, region and retries above are aruba's.

## Known gaps

- **The suite does not pass yet.** The aruba plugin does not follow the resource
  state machine the conformance scenarios assert, so failures are expected on
  status transitions rather than on provisioning.
- The auth mode installed is the chart's `single`: one API client for the whole
  cluster. The operator's multi-tenant Vault mode is out of scope for a
  conformance run.
- The operator is installed from the published chart, so a run needs network
  access to `arubacloud.github.io` and `ghcr.io`.
- **Not wired into CI.** `.github/workflows/conformance.yaml` runs the dummy plugin;
  this one needs real credentials and spends real money, so it stays manual.

## Related

- [`csp/aruba/README.md`](../../../csp/aruba/README.md) — the SECA↔Aruba mapping,
  and the plugin's own integration suite (which can run against this same cluster).
- [`../../README.md`](../../README.md) — the rest of the harness.
