# ECP test harness

This module bundles the cluster-backed test suites for ECP and the tooling to run
them. There are three kinds of test, all driven from a single `Makefile`:

| Suite | What it covers | Directory |
|-------|----------------|-----------|
| **integration** | Each component (delegator, gateway-global, gateway-regional) in **isolation**. The gateway suites test only REST↔CR translation; the delegator suite tests reconciliation. | [`integration/`](integration/) |
| **e2e** | The **whole stack in one run** — drives the SECA API on both gateways and asserts resources reconcile all the way to the delegator plugin. | [`e2e/`](e2e/) |
| **conformance** | Runs the SECA conformance suite (`secatest`) against the stack. **Plugin-generic**: point it at the dummy, aruba or ionos plugin. | [`build/conformance/`](build/conformance/), [`deploy/conformance/`](deploy/conformance/) |

The **default plugin** for the e2e and conformance stacks is the **dummy** plugin,
which logs actions instead of talking to a real cloud, so everything runs locally
on KIND.

## Directory layout

- `integration/` — the three isolated component suites (build tag `integration`).
- `e2e/` — the single end-to-end suite (build tag `e2e`).
- `internal/testenv/` — shared setup helpers (kubeconfig loading, port-forwarding)
  used by both the integration and e2e suites.
- `cmd/` — entrypoints: the `delegator` (loads the dummy or aruba plugin set) and
  the gateway start scripts.
- `build/` — a `Dockerfile` per component (delegator, gateway-global,
  gateway-regional, conformance).
- `deploy/` — Kustomize manifests per component plus `test-data` (tenant namespace,
  regions, storage SKUs).
- `scripts/` — helper scripts orchestrated by the `Makefile`.
- `context/` — **local-only** settings (kubeconfig, registry credentials). Ships
  empty and is git-ignored; see below.

## Prerequisites

[Docker](https://docs.docker.com/get-docker/),
[KIND](https://kind.sigs.k8s.io/),
[kubectl](https://kubernetes.io/docs/tasks/tools/),
[kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) and
[make](https://www.gnu.org/software/make/).

## The `context/` directory

`context/` is ignored by git and ships empty. Populate it locally to target a
remote cluster / registry instead of KIND:

- `context/kubeconfig.yaml` — if present, `deploy`/`clean`/`test`/`e2e`/`conformance`
  recipes (the non-`kind-` variants) target this cluster.
- `context/config.env` — shell exports for a remote registry, used by `make push-*`:

  ```shell
  export REGISTRY_URL="my.registry.com"
  export REGISTRY_PROJECT="my-project"
  export REGISTRY_USER="my-user"
  export REGISTRY_PASSWORD="my-password"
  ```

## Running on KIND (recommended)

```shell
make kind-start        # create the KIND cluster (once)
```

### Integration

```shell
# One component: deploy its dependencies, then run its suite.
make kind-deploy-delegator        && make kind-test-delegator
make kind-deploy-gateway-regional && make kind-test-gateway-regional
make kind-deploy-gateway-global   && make kind-test-gateway-global

# Or everything (deploy the full stack, then every suite):
make kind-deploy-all
make kind-integration
```

### End-to-end (one shot)

Builds the images, loads them into KIND, deploys the full stack with the dummy
plugin and runs the e2e suite:

```shell
make kind-e2e
```

### Conformance

```shell
# Against the dummy plugin (default):
make kind-conformance

# Pick scenarios (see scripts/conformance.sh for all CONFORMANCE_* knobs):
make kind-conformance CONFORMANCE_SCENARIOS=Storage.V1.BlockStorageLifeCycle
```

### Cleanup

```shell
make kind-clean-all    # remove deployments + CRDs, keep the cluster
make kind-stop         # delete the KIND cluster
```

## Running against a remote cluster

With `context/kubeconfig.yaml` and `context/config.env` in place, use the
non-`kind-` targets:

```shell
make build-all && make push-all         # build and push images
make e2e-deploy && make e2e             # deploy the stack, run e2e
make conformance                        # run conformance (dummy plugin)
make conformance CONFORMANCE_PLUGIN=aruba  # conform-test the aruba plugin
make clean-all                          # tear down
```

## Choosing the plugin

Both the e2e and conformance stacks reconcile with a delegator plugin, selected via
make variables that default to `dummy`. Valid values are `dummy`, `aruba` and
`ionos` — the plugin sets the delegator compiles in. `aruba` and `ionos` load and
run, but only reconcile once their backend is provisioned (the Aruba API, or
Crossplane + the IONOS provider from `csp/ionos/deploy`).

- `E2E_PLUGIN` — plugin for `make [kind-]e2e[-deploy]`.
- `CONFORMANCE_PLUGIN` — plugin for `make [kind-]conformance[-deploy]`.

Run `make help` for the full list of targets.
