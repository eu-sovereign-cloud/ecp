# ECP test harness

This module bundles the cluster-backed test suites for ECP and the tooling to run them, all driven from a single `Makefile`. There are four kinds of test:

| Suite | What it covers | Where |
|-------|----------------|-------|
| **integration** | Each component (delegator, gateway-global, gateway-regional) in **isolation**. The gateway suites test only REST↔CR translation; the delegator suite tests reconciliation. | [`integration/`](integration/) |
| **e2e** | The **whole stack in one run** — drives the SECA API on both gateways and asserts resources reconcile all the way to the delegator plugin. Single cluster. | [`e2e/`](e2e/) |
| **multicluster e2e** | The **split topology** — global gateway in one cluster, regional gateway + delegator in another, joined only by the Region CR the global gateway advertises. | [`e2e/multicluster/`](e2e/multicluster/) |
| **conformance** | Runs the SECA conformance suite (`secatest`) against the stack. | [`internal/build/conformance/`](internal/build/conformance/), [`internal/deploy/conformance/`](internal/deploy/conformance/) |

## Layout

Only the test suites live at the top level; everything they build on sits under `internal/` (nothing here is imported by other modules).

```
test/
  Makefile  README.md  go.mod  go.sum
  integration/          # isolated component suites (build tag `integration`)
  e2e/                  # single end-to-end suite (build tag `e2e`)
    multicluster/       # two-cluster suite (build tag `multicluster`)
  conformance/
    ionos/              # IONOS real-backend conformance (split global/regional demo)
      cluster/          #   manifests for the demo's two clusters
      scripts/          #   cluster setup / teardown for the demo
    aruba/              # placeholder for an aruba real-backend harness
  internal/
    testenv/            # shared kubeconfig + port-forward helpers (Go)
    authhelper/         # shared auth test helpers (build tag `authhelper`)
    cmd/                # entrypoints: delegator + benchreport
    build/              # a Dockerfile per component (incl. conformance runner)
    deploy/             # per-component deployment inputs: chart values, or
                        # Kustomize manifests for what no chart deploys
    kind-config/        # KIND cluster configs (multicluster regional port mapping)
    scripts/            # helper scripts orchestrated by the Makefile
    context/            # LOCAL-ONLY settings (git-ignored, ships empty)
```

## The plugin model: one-shot vs multi-phase

Both the e2e and conformance stacks reconcile with a **delegator plugin**. The delegator compiles in three plugin sets — `dummy`, `aruba`, `ionos` — selected by `E2E_PLUGIN` / `CONFORMANCE_PLUGIN` (default `dummy`). What differs is the **backend** each needs, and that dictates how you run it:

| Plugin | Backend | How to run |
|--------|---------|------------|
| **dummy** | none (logs actions) | **one-shot** — fully self-contained on KIND |
| **aruba** | `arubacloud-resource-operator` + Aruba creds | **multi-phase** — deploy → install backend → run |
| **ionos** | Crossplane + IONOS provider + token | **multi-phase**, or the bespoke `conformance/ionos` real-backend run |

Only **dummy** is self-contained, so the one-shot targets (`kind-integration`, `kind-e2e`, `kind-test-all`, `kind-conformance`) always use dummy. aruba/ionos can't run in one command — their resources never reconcile until their backend exists — so they use the two-phase `*-deploy` → (provision backend) → run flow described below. This is why `E2E_PLUGIN` / `CONFORMANCE_PLUGIN` are honoured on the `*-deploy` targets but not on the one-shot targets.

## One stack, every suite

Integration and e2e run against the **same deployment**: `test-data`, both gateways and the delegator, all with the same authentication plugin (`AUTH_PLUGIN`, default `dummy`). `deploy-stack` / `kind-deploy-stack` is that deployment; the one-shot targets build, load and deploy it for you.

| Target | Runs |
|--------|------|
| `[kind-]integration` | every integration suite |
| `[kind-]integration-<component>` | one integration suite (`delegator`, `gateway-global`, `gateway-regional`) |
| `[kind-]e2e` | the e2e suite |
| `[kind-]test-all` | **everything** — integration + e2e |

The `kind-` variants are one-shot (build → load → deploy → run); the plain ones run the suite against an already-deployed cluster.

The multicluster suite is **not** part of `test-all`: it needs its own pair of clusters, so it has a separate target set (see [Multicluster e2e](#multicluster-e2e-two-clusters)).

## How components are deployed

The gateways and the delegator are deployed from the Helm charts this repository publishes — [`chart/`](../chart) and [`chart-delegator/`](../chart-delegator) — not from test-only manifests. `internal/deploy/<component>/values.yaml` holds what the test stack pins; `deploy.sh` supplies the rest (image, pull policy, auth plugin, delegator plugin) with `--set` and runs `helm upgrade --install --wait`.

The point is that there is one deployment path, not two: a template that breaks, an RBAC rule that goes missing, or a value that reaches no flag fails a test run here rather than a user's first `helm install`. Keeping the charts correct is therefore part of keeping the suites green.

| Component | Deployed by | Release |
|-----------|-------------|---------|
| `gateway-global` | `chart/`, other gateway disabled | `ecp-global` |
| `gateway-regional` | `chart/`, other gateway disabled | `ecp-regional` |
| `delegator` | `chart-delegator/`, `plugin` from `PLUGIN_TYPE` | `ecp-delegator` |
| `test-data` | kustomize — fixture CRs, nothing anyone installs | — |
| `conformance` | kustomize — the secatest runner | — |

Two consequences worth knowing:

- **Names come from the chart.** The Deployments and Services are `ecp-global-gateway-global`, `ecp-regional-gateway-regional` and `ecp-delegator`, not the old `*-depl` / `*-svc`. The suites port-forward by pod label (`app=gateway-global`), which the chart still sets, so they are unaffected; anything dialling a gateway by DNS is not, and `internal/scripts/common.sh` holds the service names for it (`test-data/regions.yaml` carries the same ones).
- **The delegator's RBAC follows its plugin.** `chart-delegator` grants exactly the controller set `plugin` loads, so adding a resource to a plugin means adding its rules to that plugin's branch in `chart-delegator/templates/rbac.yaml`.

To deploy the same stack by hand, or to install it anywhere real, use the charts directly — see [`chart/README.md`](../chart/README.md).

## Prerequisites

[Docker](https://docs.docker.com/get-docker/),
[KIND](https://kind.sigs.k8s.io/),
[kubectl](https://kubernetes.io/docs/tasks/tools/),
[Helm](https://helm.sh/docs/intro/install/),
[kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) and
[make](https://www.gnu.org/software/make/).

## Local context (`internal/context/`)

`internal/context/` is git-ignored and ships empty. Populate it locally to target a
remote cluster / registry instead of KIND:

- `internal/context/kubeconfig.yaml` — if present, the non-`kind-` recipes
  (`deploy`/`clean`/`test`/`e2e`/`conformance`) target this cluster.
- `internal/context/config.env` — shell exports for a remote registry:

  ```shell
  export REGISTRY_URL="my.registry.com"
  export REGISTRY_PROJECT="my-project"
  export REGISTRY_USER="my-user"
  export REGISTRY_PASSWORD="my-password"
  ```

  `make push-*` logs in with these. `deploy`/`conformance` also mint an
  image-pull secret from `REGISTRY_USER`/`REGISTRY_PASSWORD`/`REGISTRY_URL` and
  attach it to the pods, so a remote cluster can pull from a private registry.
  No credential is committed to the repo; leave these unset for KIND (images are
  side-loaded) or a public registry (nothing to authenticate).

## Running on KIND (dummy plugin)

```shell
make kind-start        # create the KIND cluster (once)
```

### Integration

```shell
# One component: deploy its dependencies, then run its suite.
make kind-deploy-delegator        && make kind-integration-delegator
make kind-deploy-gateway-regional && make kind-integration-gateway-regional
make kind-deploy-gateway-global   && make kind-integration-gateway-global

# Or one shot (build, load, deploy the full stack, then every integration suite):
make kind-integration
```

### End-to-end (one shot)

Builds the images, loads them into KIND, deploys the full stack with the dummy plugin and runs the e2e suite:

```shell
make kind-e2e
```

### Everything (one shot)

Same stack, both suites — integration and e2e:

```shell
make kind-test-all
make kind-test-all AUTH_PLUGIN=jwt   # …with the gateways verifying signed JWTs
```

### Conformance (one shot)

```shell
make kind-conformance

# Pick scenarios (see internal/scripts/conformance.sh for all CONFORMANCE_* knobs):
make kind-conformance CONFORMANCE_SCENARIOS=Storage.V1.BlockStorageLifeCycle
```

## Multicluster e2e (two clusters)

Everything above runs both gateways in **one** cluster, where the region catalog's
provider URLs are an unasserted fixture pointing at in-cluster DNS. This suite runs the
real split topology instead: the global gateway in `e2e-global`, the regional gateway
and delegator in `e2e-regional`.

The suite is handed **only** the global endpoint plus a kubeconfig context per cluster.
It discovers the regional API by reading the provider URLs off the Region CR, so a
broken registration fails the run instead of passing on a fixture. It then asserts the
workspace CR it creates lands in the regional cluster and **not** the global one — the
resource genuinely crossed a cluster boundary.

```shell
make kind-multicluster-e2e                   # one-shot: create both clusters, deploy, run
make kind-multicluster-e2e AUTH_PLUGIN=jwt   # …with the gateways verifying signed JWTs

# Or step by step:
make kind-multicluster-start                 # create both clusters
make kind-multicluster-stack                 # deploy the split stack + register the region
make multicluster-e2e                        # run the suite against an already-deployed pair
make kind-multicluster-stop                  # delete both clusters
```

The regional gateway is exposed for cross-cluster reach the same way everything else is
deployed — through the chart. `kind-multicluster-stack` layers
[`gateway-regional/multicluster-values.yaml`](internal/deploy/gateway-regional/multicluster-values.yaml)
onto the deploy (via `DEPLOY_VALUES`), which sets `service.type=NodePort` and pins
`service.nodePort`. `register-region.sh` then performs the join: it reads that NodePort
back off the chart-created Service, confirms the pin, and writes a Region CR advertising
the address into the *global* cluster, overwriting the static `test-data` fixture of the
same name.

| Variable | Default | Description |
|----------|---------|-------------|
| `MULTICLUSTER_GLOBAL_CLUSTER` | `e2e-global` | KIND cluster for the global gateway. |
| `MULTICLUSTER_REGIONAL_CLUSTER` | `e2e-regional` | KIND cluster for the regional gateway + delegator. |
| `MULTICLUSTER_GLOBAL_CONTEXT` | `kind-$(MULTICLUSTER_GLOBAL_CLUSTER)` | Context the scripts and suite use for the global cluster. |
| `MULTICLUSTER_REGIONAL_CONTEXT` | `kind-$(MULTICLUSTER_REGIONAL_CLUSTER)` | Context for the regional cluster. |
| `MULTICLUSTER_REGION` | `itbg-bergamo` | Region name registered. Must match the regional gateway's `REGION` env. |
| `MULTICLUSTER_REGIONAL_NODE_PORT` | `30080` | Regional gateway NodePort. Must match the `extraPortMappings` entry in `internal/kind-config/regional-cluster.yaml`. |
| `MULTICLUSTER_ADVERTISE_HOST` | `127.0.0.1` | Host advertised in the Region CR. |

> **Why a published host port rather than the KIND node IP.** The suite reaches the
> regional gateway at whatever address the Region CR advertises, so that address must be
> reachable from the machine running the tests. A KIND node's own bridge IP
> (`172.18.x.x`) works on a native Linux host but **not** from a WSL2 distro against
> Docker Desktop, where the containers live in a separate VM network namespace.
> `internal/kind-config/regional-cluster.yaml` publishes the NodePort to the host so
> `127.0.0.1:30080` behaves the same everywhere. Set `MULTICLUSTER_ADVERTISE_HOST` to the
> node IP if you would rather exercise the bridge path on native Linux.

Because two clusters roughly double the wall time for a topology that changes rarely,
this suite is best run nightly or on changes under `internal/deploy/` rather than on
every PR.

### Cleanup

```shell
make kind-clean-all    # remove deployments + CRDs, keep the cluster
make kind-stop         # delete the KIND cluster
```

## Running the aruba / ionos plugins (multi-phase)

These need their backend, so the run is three steps: **deploy the stack with the plugin → provision the backend → run the suite.**

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

Swap `conformance-deploy`/`conformance` for `deploy-stack E2E_PLUGIN=<plugin>`/`e2e` to run the e2e suite instead. See [`conformance/aruba/`](conformance/aruba/) for the aruba backend caveats.

### IONOS real-backend run (`conformance/ionos`)

IONOS also has a fully-bundled path that stands up the multi-cluster demo, installs Crossplane + the provider, and runs `secatest` — reachable from this Makefile:

```shell
make conformance-ionos-scaffolding   # build images, create demo clusters, install plugin
make conformance-ionos               # run secatest via NodePort
make conformance-ionos-clean         # tear down
```

Its clusters are **not** the multicluster e2e pair above: the demo stands up its own
`global` / `regional` clusters (kubeconfigs under `~/.kube/multi-cluster-demo/`) with an
auth-disabled, `seca`-tenant topology that `secatest` expects. The manifests and setup
scripts for it live in [`conformance/ionos/cluster/`](conformance/ionos/cluster/) and
[`conformance/ionos/scripts/`](conformance/ionos/scripts/).

## Running against a remote cluster

With `internal/context/kubeconfig.yaml` and `internal/context/config.env` in place, use the non-`kind-` targets:

```shell
make build-all && make push-all         # build and push images
make deploy-stack && make test-all      # deploy the dummy stack, run integration + e2e
make conformance                        # run conformance
make clean-all                          # tear down
```

Run `make help` for the full list of targets.

## Authentication & Authorization in e2e

The test stack deploys the gateways with the Dummy authenticator and SECA RBAC enabled (the chart's own default is auth **off**, mirroring the binary; `internal/deploy/gateway-values.yaml` turns it on). Auth behaviour is configured by the chart, which passes every setting to the binary as a command-line flag — see [Gateway auth values](#gateway-auth-values).

### Which authenticator runs where

A gateway serves one authentication plugin at a time (`--auth-plugin`), and **both gateways are deployed with the same one** — the value of `AUTH_PLUGIN` (`dummy`, the default, or `jwt`). `deploy.sh` passes it to both releases as `auth.plugin`, and the suites read the same variable (`authhelper.Token`) to mint matching tokens, so any suite runs against any stack:

```sh
make kind-test-all                   # every suite against dummy tokens
make kind-test-all AUTH_PLUGIN=jwt   # every suite against signed JWTs
```

Deploy and test with the **same** value: a dummy token sent to a jwt-backed gateway (or the reverse) is a 401 on every request. Plugin-specific cases skip themselves under the other plugin — `TestJWTAuthn` (e2e) needs `jwt`, the wrong-password case of `TestAuthn` needs `dummy`.

`deploy.sh` runs `helm upgrade --install --wait`, so a suite started right after a redeploy never port-forwards to a terminating pod still serving the previous plugin.

### Gateway auth values

Set in [`internal/deploy/gateway-values.yaml`](internal/deploy/gateway-values.yaml), shared by both gateway releases. The chart renders each one into a flag on the container's `args` — there is no env-var path, so overriding any of these means editing the values file (or adding a `--set` in `deploy.sh`), not setting a variable in your shell. `AUTH_PLUGIN` is the exception: it is a Makefile variable precisely because the suites need to agree with it.

| Value | Test setting | Flag | Description |
|-------|--------------|------|-------------|
| `auth.enabled` | `true` | `--auth-enabled` | Set to `false` to run the gateways without any auth (unauthenticated mode). |
| `auth.plugin` | `dummy` | `--auth-plugin` | Authenticator to run: `dummy` or `jwt`. Overridden per deployment from `AUTH_PLUGIN` and read by the suites, so it is a single setting for the whole stack. |
| `auth.jwt.signingMethod` | `ES256` | `--jwt-signing-method` | Expected JWT `alg` when the plugin is `jwt`. Must match what the suite signs with (`authhelper.JWTSigningMethod`). |
| `auth.jwt.key` | committed fixture | `--jwt-secret` | Verification key, rendered into a Secret and mounted at `/etc/ecp/jwt/jwt.pub`. |
| `auth.authz.enabled` | `true` | `--authz-enabled` | Set to `false` for authn-only (identity checked, no RBAC). Requires `auth.enabled`. |
| `auth.authz.impl` | `cached` | `--authz-cache` | `cached` uses the informer-backed checker (zero K8s round-trips on hot path); `direct` uses the per-request reader (2 K8s List calls per request). |
| `auth.authz.skipProviders` | `seca.region` | `--authz-skip-providers` | Comma-separated provider IDs served authn-only (no RBAC check, no token down-scoping). The region catalog is tenant-less by spec. |
| `auth.dummyUsers.users` | 7 fixture users | `--dummy-auth-users` | username → password map, rendered into a Secret and mounted at `/etc/ecp/auth/users.json`. |

### Test-side env vars

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_PLUGIN` | `dummy` | Token format the suites mint — the same variable the stack is deployed with, exported by the Makefile so the two cannot drift. |
| `E2E_AUTH_ENABLED` | `true` (implicit) | Set to `false` to skip all auth-specific test assertions; useful when running against a gateway deployed with `AUTH_ENABLED=false`. |
| `E2E_BENCH` | _(unset)_ | Set to `1` to run the `TestBench` load workload (skipped by default). |
| `E2E_BENCH_REQUESTS` | `500` | Number of requests fired by `TestBench`. |

### Test fixtures: subjects, users, and assignments

The files in `internal/deploy/test-data/` define the RBAC state used by the auth tests. Roles are **not** carried by the token — each subject's roles come entirely from the RoleAssignment named below (the token carries only the subject and an optional down-scope). The table maps the token subject to the RoleAssignment that covers them and the net access they should receive.

The subject is the dummy token's `username` or the JWT's `sub` claim; both plugins feed the same `Identity.Subject`, so **every row applies unchanged to either plugin** — the password column is simply unused for JWTs, which are trusted by signature instead.

| Subject | Password | RoleAssignment | Roles (from assignment) | Scope | Expected result |
|---------|----------|----------------|-------------------------|-------|-----------------|
| `admin` | `e2e-admin-pass` | `ra-admin` | `e2e-admin` (all providers, all resources) | all | ✅ All operations |
| `alice` | `alice-pass` | `ra-alice-region-viewer` | `e2e-region-viewer` (`seca.region`) | `test-tenant` | ❌ cross-provider ops (her only role covers `seca.region`, which is authn-only anyway) |
| `bob` | `bob-pass` | `ra-bob-scoped` | `e2e-storage-viewer` (`seca.storage` get/list: `block-storages`, `images`, `storage-skus`) | `test-tenant` + region `itbg-bergamo` | ✅ List block-storages in that region; ❌ other regions (incl. token down-scoped elsewhere) |
| `carol` | `carol-pass` | `ra-multi-subject` | `e2e-workspace-editor` | `test-tenant` | ✅ Workspace CRUD |
| `dave` | `dave-pass` | `ra-multi-subject` | `e2e-workspace-editor` | `test-tenant` | ✅ Workspace CRUD |
| `erin` | `erin-pass` | `ra-wrong-tenant` | `e2e-admin` scoped to `other-tenant` | `other-tenant` | ❌ admin ops in `test-tenant` (out of scope) |
| `nobody` | `nobody-pass` | _(none)_ | _(none)_ | — | ✅ List regions (authn-only provider); ❌ everything RBAC-governed (e.g. `seca.authorization` → 403) |

The region catalog (`seca.region`) is served **authn-only**: `--authz-skip-providers` defaults to `seca.region` because the region resource is tenant-less by spec, so tenant-scoped RBAC cannot govern it. Every authenticated caller — including `nobody`, who has no RoleAssignment at all — can list regions. A genuine 403 therefore requires an RBAC-governed provider (the tests use `seca.authorization`). Down-scoping tests present a broad identity (e.g. `admin`, or `bob` in-region) with a narrow token `scope` and assert the request is denied outside the cap.

> ⚠️ The Dummy authenticator performs no signature verification — any caller who knows a valid username+password can impersonate that subject. These credentials must never be used in production.

### JWT test fixtures

`internal/authhelper` mints the tokens the JWT-backed gateways accept:

| Helper | Use |
|--------|-----|
| `Token(user, password, scope)` | The token for the **deployed** plugin — a signed JWT under `AUTH_PLUGIN=jwt`, a dummy token otherwise. Backs `AdminEditor` / `IdentityEditor` / `ScopedEditor`, so the suites are plugin-agnostic. |
| `JWTAuth()` | Whether the stack runs the jwt plugin; plugin-specific tests skip on it. |
| `JWTKey()` | The fixture ES256 private key. Pass a freshly generated key instead to forge a token the gateway must reject. |
| `SignJWT(key, subject, scope, exp)` | Sign a token for a subject, with an optional down-scope and an explicit expiry (pass a past time for an expired token). |

The key pair is a committed fixture: the private half is a constant in `authhelper`, the public half is `auth.jwt.key` in `internal/deploy/gateway-values.yaml`. The chart renders it into a Secret, mounts it at `/etc/ecp/jwt/jwt.pub` and passes that path as `--jwt-secret`. It is a Secret rather than a ConfigMap because the same value serves `signingMethod: HS*`, where the file is the shared HMAC secret that mints tokens — see [Verification key](../doc/AUTH.md#verification-key---jwt-secret). To rotate it, regenerate both halves and update both places:

```sh
openssl ecparam -name prime256v1 -genkey -noout -out jwt-key.pem
openssl pkcs8 -topk8 -nocrypt -in jwt-key.pem   # private half → authhelper constant
openssl ec -in jwt-key.pem -pubout              # public half  → gateway-values.yaml
```

> ⚠️ This key pair is a test fixture whose private half is published in this repository — anyone can mint a token the e2e gateway accepts. Like the dummy credentials, it must never be used in production.

### Running auth tests

Auth tests are automatically included when running the normal test suite against a cluster with `AUTH_ENABLED=true` (the default):

```sh
make kind-integration-gateway-global    # dummy tokens (the default)
make kind-test-all AUTH_PLUGIN=jwt      # the same suites, signed JWTs
```

`TestJWTAuthn` (`e2e/jwt_test.go`) is the JWT-specific suite: valid/expired/unsigned-by-us tokens, algorithm-confusion attempts, dummy tokens rejected by the JWT gateway, and the `sub`/`scope` claims driving RBAC. It runs only under `AUTH_PLUGIN=jwt` and is the only test that covers the `--jwt-secret` file → `ParseVerifyKey` → authenticator wiring, since the unit tests build the authenticator from an already-parsed key.

To skip auth assertions (e.g. against an auth-disabled gateway):

```sh
E2E_AUTH_ENABLED=false make kind-integration-gateway-global
```

---

## Benchmarking the Auth Middleware

The `TestBench` load test fires authenticated requests to populate the Prometheus `ecp_gateway_*_duration_seconds` histograms on the deployed gateway. The `benchreport` tool then scrapes `/metrics`, computes latency statistics, and writes a markdown report.

### Quick start

```sh
# 1. Deploy with cached checker (the default)
make kind-deploy-gateway-global   # AUTHZ_IMPL=cached by default

# 2. Fire the load workload
make kind-bench                   # E2E_BENCH=1; default 500 requests

# 3. Scrape metrics and generate the report
IMPL_TAG=cached make report       # writes internal/report/REPORT.md

# 4. Redeploy with the direct checker (helm upgrades the release in place —
#    no need to clean first)
AUTHZ_IMPL=direct make kind-deploy-gateway-global

# 5. Fire another load workload and save a second snapshot
E2E_BENCH_REQUESTS=500 make kind-bench
IMPL_TAG=direct SNAP_FILE=internal/report/snap-direct.txt make report

# 6. Merge both snapshots into one comparison report
go run ./internal/cmd/benchreport \
    --impl=cached --metrics-file=internal/report/snap.txt \
    --impl=direct --metrics-file=internal/report/snap-direct.txt \
    --out=internal/report/REPORT.md
```

### Reading the report

`internal/report/REPORT.md` contains three latency tables — one per histogram — with rows for each `impl/label` combination and columns:

| Column | Meaning |
|--------|---------|
| `count` | Total number of observations. |
| `avg (ms)` | Arithmetic mean latency in milliseconds. |
| `p50 (ms)` | Median latency (50th percentile), interpolated from buckets. |
| `p90 (ms)` | 90th-percentile latency. |
| `p99 (ms)` | 99th-percentile latency. |

Expected comparison pattern:

- `ecp_gateway_rbac_fetch_duration_seconds{impl="cached"}` p99 should be orders of magnitude lower than `impl="direct"` (in-memory read vs K8s List).
- `ecp_gateway_authz_check_duration_seconds` should mirror the fetch delta.
- `ecp_gateway_auth_middleware_duration_seconds` differences reflect the checker cost amortised over the full request (provider handler is a constant).