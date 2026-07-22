# ecp-delegator

Helm chart for the ECP **delegator with the Aruba Cloud plugin** — the
controller that watches the ECP Custom Resources (written by the regional
gateway) and reconciles them by writing `arubacloud.com` CRs, which the
third-party
[arubacloud-resource-operator](https://github.com/Arubacloud/arubacloud-resource-operator)
acts on against Aruba Cloud.

The delegator binary itself is plugin-generic; this chart is the `csp/aruba`
specialization and pins `PLUGIN=aruba` — its RBAC covers exactly the aruba
plugin set (workspace + block-storage controllers). Other CSPs ship their own
deploy tooling (e.g. `csp/ionos/deploy`).

## Prerequisites

- The ECP CRDs, installed by the [`ecp`](../../../chart) chart (or
  `kubectl apply -f chart/crds/`).
- Install into the **same cluster as the regional gateway** — the delegator
  reconciles the CRs that gateway writes.
- The arubacloud-resource-operator plus its Aruba credentials, installed
  **out of band** (no in-repo tooling yet — see
  `test/conformance/aruba/README.md`). Until it is present, resources stay
  pending.

## Installing

The delegator image is published to `ghcr.io/eu-sovereign-cloud/ecp/delegator`
on every `v*` tag (see `.github/workflows/image-release.yaml`). One image
serves every CSP; this chart pins `PLUGIN=aruba`.

```bash
helm install ecp-delegator chart-delegator \
  --namespace ecp --create-namespace
```

## Values

See [values.yaml](values.yaml) for the full commented list. The notable ones:

| Key | Default | Notes |
|-----|---------|-------|
| `image.repository` | `ghcr.io/eu-sovereign-cloud/ecp/delegator` | Override only to mirror the image into your own registry |
| `replicaCount` | `1` | Keep at 1: the delegator runs without leader election |
| `rbac.create` | `true` | ClusterRole scoped to the aruba plugin's controller set |
