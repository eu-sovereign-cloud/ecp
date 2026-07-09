# ECP test harness

This module bundles the cluster-backed test suites for ECP and the tooling to run
them, all driven from a single `Makefile`. There are three kinds of test:

| Suite | What it covers | Where |
|-------|----------------|-------|
| **integration** | Each component (delegator, gateway-global, gateway-regional) in **isolation**. The gateway suites test only REST↔CR translation; the delegator suite tests reconciliation. | [`integration/`](integration/) |
| **e2e** | The **whole stack in one run** — drives the SECA API on both gateways and asserts resources reconcile all the way to the delegator plugin. | [`e2e/`](e2e/) |
| **conformance** | Runs the SECA conformance suite (`secatest`) against the stack. | [`internal/build/conformance/`](internal/build/conformance/), [`internal/deploy/conformance/`](internal/deploy/conformance/) |

## Layout

Only the test suites live at the top level; everything they build on sits under
`internal/` (nothing here is imported by other modules).

```
test/
  Makefile  README.md  go.mod  go.sum
  integration/          # isolated component suites (build tag `integration`)
  e2e/                  # single end-to-end suite (build tag `e2e`)
  conformance/
    ionos/              # IONOS real-backend conformance (multi-cluster demo)
    aruba/              # placeholder for an aruba real-backend harness
  internal/
    testenv/            # shared kubeconfig + port-forward helpers (Go)
    cmd/                # entrypoints: delegator + gateway start scripts
    build/              # a Dockerfile per component (incl. conformance runner)
    deploy/             # Kustomize manifests per component + test-data
    scripts/            # helper scripts orchestrated by the Makefile
    context/            # LOCAL-ONLY settings (git-ignored, ships empty)
```

## The plugin model: one-shot vs multi-phase

Both the e2e and conformance stacks reconcile with a **delegator plugin**. The
delegator compiles in three plugin sets — `dummy`, `aruba`, `ionos` — selected by
`E2E_PLUGIN` / `CONFORMANCE_PLUGIN` (default `dummy`). What differs is the
**backend** each needs, and that dictates how you run it:

| Plugin | Backend | How to run |
|--------|---------|------------|
| **dummy** | none (logs actions) | **one-shot** — fully self-contained on KIND |
| **aruba** | `arubacloud-resource-operator` + Aruba creds | **multi-phase** — deploy → install backend → run |
| **ionos** | Crossplane + IONOS provider + token | **multi-phase**, or the bespoke `conformance/ionos` real-backend run |

Only **dummy** is self-contained, so the one-shot targets (`kind-e2e`,
`kind-conformance`) always use dummy. aruba/ionos can't run in one command — their
resources never reconcile until their backend exists — so they use the two-phase
`*-deploy` → (provision backend) → run flow described below. This is why
`E2E_PLUGIN` / `CONFORMANCE_PLUGIN` are honoured on the `*-deploy` targets but not
on the one-shot targets.

## Prerequisites

[Docker](https://docs.docker.com/get-docker/),
[KIND](https://kind.sigs.k8s.io/),
[kubectl](https://kubernetes.io/docs/tasks/tools/),
[kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) and
[make](https://www.gnu.org/software/make/).

## Local context (`internal/context/`)

`internal/context/` is git-ignored and ships empty. Populate it locally to target a
remote cluster / registry instead of KIND:

- `internal/context/kubeconfig.yaml` — if present, the non-`kind-` recipes
  (`deploy`/`clean`/`test`/`e2e`/`conformance`) target this cluster.
- `internal/context/config.env` — shell exports for a remote registry, used by
  `make push-*`:

  ```shell
  export REGISTRY_URL="my.registry.com"
  export REGISTRY_PROJECT="my-project"
  export REGISTRY_USER="my-user"
  export REGISTRY_PASSWORD="my-password"
  ```

## Running on KIND (dummy plugin)

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

### Conformance (one shot)

```shell
make kind-conformance

# Pick scenarios (see internal/scripts/conformance.sh for all CONFORMANCE_* knobs):
make kind-conformance CONFORMANCE_SCENARIOS=Storage.V1.BlockStorageLifeCycle
```

### Cleanup

```shell
make kind-clean-all    # remove deployments + CRDs, keep the cluster
make kind-stop         # delete the KIND cluster
```

## Running the aruba / ionos plugins (multi-phase)

These need their backend, so the run is three steps: **deploy the stack with the
plugin → provision the backend → run the suite.**

```shell
# aruba
make conformance-deploy CONFORMANCE_PLUGIN=aruba   # deploy stack + aruba plugin
#   ... install the arubacloud-resource-operator + Aruba credentials ...
make conformance                                   # run against the deployed stack

# ionos (single-cluster, delegator plugin)
make conformance-deploy CONFORMANCE_PLUGIN=ionos
#   ... install Crossplane + the IONOS provider (see csp/ionos/deploy) ...
make conformance
```

Swap `conformance-deploy`/`conformance` for `e2e-deploy`/`e2e` to run the e2e suite
instead. See [`conformance/aruba/`](conformance/aruba/) for the aruba backend
caveats.

### IONOS real-backend run (`conformance/ionos`)

IONOS also has a fully-bundled path that stands up the multi-cluster demo, installs
Crossplane + the provider, and runs `secatest` — reachable from this Makefile:

```shell
make conformance-ionos-scaffolding   # build images, create demo clusters, install plugin
make conformance-ionos               # run secatest via NodePort
make conformance-ionos-clean         # tear down
```

## Running against a remote cluster

With `internal/context/kubeconfig.yaml` and `internal/context/config.env` in place,
use the non-`kind-` targets:

```shell
make build-all && make push-all         # build and push images
make e2e-deploy && make e2e             # deploy the dummy stack, run e2e
make conformance                        # run conformance
make clean-all                          # tear down
```

Run `make help` for the full list of targets.
