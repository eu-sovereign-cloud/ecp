# k6 functional API load tests

Black-box **API journeys** run with [k6](https://k6.io/) against a **deployed**
ECP stack (global + regional gateways). Each journey is a short user-shaped
path through the SECA HTTP API (auth headers, tenant paths, create/get/delete).

| This harness                             | Not this harness                                    |
|------------------------------------------|-----------------------------------------------------|
| HTTP only, outside the cluster process   | Go unit / integration / e2e suites                  |
| Journey scripts under `journeys/`        | Component isolation or CR reconciliation asserts    |
| Thresholds on checks + `http_req_failed` | Go `TestBench` + `benchreport` (Prometheus latency) |

Go **e2e** still owns full-stack reconcile guarantees. Go **bench** warms
gateway histograms for `make report`. k6 owns portable, black-box functional
load journeys that scale later to multi-VU profiles without rewriting the Go
suites.

## Prerequisites

1. **k6** — native binary or Docker. `make k6-start` resolves one.
2. **kubectl** + cluster access — for `ensure-tenant` and stack deploy.
3. **Deployed test stack** — same stack as integration/e2e. On KIND, create
   the cluster once, then build/load/deploy:

   ```bash
   make -C test kind-start   # once; creates cluster e2e-cluster
   make -C test kind-stack   # build images, load into KIND, deploy stack
   # or: make -C test deploy-stack  (existing non-KIND cluster)
   ```

   `kind-stack` does **not** create the cluster; run `kind-start` first if
   `e2e-cluster` is missing.

4. **Gateway base URLs** — no auto port-forward. Point at real endpoints or
   local forwards, then export:

   ```bash
   export BASE_URL_GLOBAL=http://127.0.0.1:8080
   export BASE_URL_REGIONAL=http://127.0.0.1:8081
   ```

   Example port-forwards (namespace default `e2e-ecp`). Gateway Services
   expose **port 80** (pods still listen on 8080):

   ```bash
   kubectl -n e2e-ecp port-forward svc/ecp-global-gateway-global 8080:80 &
   kubectl -n e2e-ecp port-forward svc/ecp-regional-gateway-regional 8081:80 &
   ```

   Or forward to the Deployment/container port:

   ```bash
   kubectl -n e2e-ecp port-forward deploy/ecp-global-gateway-global 8080:8080 &
   kubectl -n e2e-ecp port-forward deploy/ecp-regional-gateway-regional 8081:8080 &
   ```

   Service names can differ with chart/release overrides; pods are selectable
   with `app.kubernetes.io/component=gateway-global` /
   `app.kubernetes.io/component=gateway-regional` if needed.

## Full happy path (KIND → journeys)

```bash
# 1. KIND cluster (once)
make -C test kind-start

# 2. Stack (images + load + deploy + test-data + dummy plugin)
make -C test kind-stack

# 3. Port-forward both gateways (local:remote — Services use port 80)
kubectl -n e2e-ecp port-forward svc/ecp-global-gateway-global 8080:80 &
kubectl -n e2e-ecp port-forward svc/ecp-regional-gateway-regional 8081:80 &
export BASE_URL_GLOBAL=http://127.0.0.1:8080
export BASE_URL_REGIONAL=http://127.0.0.1:8081

# 4. Tooling + offline check
make -C test/load k6-start
make -C test/load selfcheck

# 5. Journeys (ensure-tenant runs as a dependency)
make -C test/load smoke
make -C test/load create-workspace

# Optional: wait for workspace Active (delegator)
WAIT_ACTIVE=1 make -C test/load create-workspace
```

From the parent test Makefile:

```bash
make -C test load-smoke
make -C test load-create-workspace
# BASE_URL_* still required in the environment
```

## GKE load-test path

After gateways (+ optional dummy) are on GKE via `setup_gke.sh` and public LBs
exist (`global.test.ecociel.com` / `regional.test.ecociel.com`):

```bash
# 1. Seed roles, SKUs, regions; rewrite provider URLs to public bases
make -C test/load gke-fixtures
# overrides: GKE_SYSTEM_NAMESPACE=ecp BASE_URL_GLOBAL=… BASE_URL_REGIONAL=…

# 2. Point journeys at the public gateways (no Makefile default — see below)
export BASE_URL_GLOBAL=http://global.test.ecociel.com
export BASE_URL_REGIONAL=http://regional.test.ecociel.com
export SYSTEM_NAMESPACE=ecp
export E2E_AUTH_ENABLED=false   # true + AUTH_PASS when auth is on

# 3. Journeys
make -C test/load smoke
make -C test/load create-workspace
```

Parent aliases: `make -C test load-gke-fixtures`, `load-smoke`, …

`gke-fixtures` applies `test-data` into `GKE_SYSTEM_NAMESPACE` (default `ecp`),
creates Region `europe-west1` with public regional provider URLs, rewrites
in-cluster Service DNS on other Region CRs, and runs `ensure-tenant`.

## Local dev-cluster path (`*-dev` targets, no port-forward)

Every journey target has a `-dev` variant (`smoke-dev`, `create-workspace-dev`,
`stepwise-dev`, `stress-dev`, `throughput-dev`, `tf-stress-dev`,
`tf-net-storage-dev`) that runs against the **ionos conformance suite's
dev-cluster setup** — two standalone KIND clusters named `global`/`regional`, distinct from
`test`'s single `e2e-cluster` — with zero manual setup:

```bash
make -C test/load smoke-dev
```

Each `-dev` target:

1. Runs `ensure-dev-clusters` — if the `global`/`regional` KIND clusters don't
   already exist, builds the gateway images and bootstraps them
   (`test/conformance/ionos/scripts/setup-clusters.sh`; first run takes a few minutes,
   later runs no-op).
2. Discovers `BASE_URL_GLOBAL`/`BASE_URL_REGIONAL` from each cluster's
   NodePort Service directly (`gateway/scripts/dev-url.sh`) — both services
   are `type: NodePort`, reachable at
   `http://<control-plane-container-ip>:<nodePort>` with no port-forward.
3. Points `KUBECONFIG` at the **regional** cluster (where tenant/workspace CRs
   live) and sets `E2E_TENANT=seca` — the tenant these dev clusters seed a
   namespace + `StorageSKU`/`InstanceSKU` fixtures for
   (`test/conformance/ionos/cluster/regional/`), **not** the GKE/e2e-cluster
   default `test-tenant`.
4. Re-invokes the plain target (e.g. `smoke`) with those as environment
   overrides.

`DEV_KUBECONFIG_DIR` (`~/.kube/multi-cluster-demo`) is fixed to match what
`setup-clusters.sh` itself hardcodes — it is **not** meant to be
overridden independently.

```bash
kind delete cluster --name global && kind delete cluster --name regional
```

## Make targets

| Target | What it does |
|--------|----------------|
| `k6-start` | Resolve native k6 or Docker image; write `scripts/.k6-cmd` |
| `k6-version` | Print resolved k6 version |
| `hello` | Print config (no network, URLs optional) |
| `selfcheck` | Offline config/auth checks + `options/smoke.json` |
| `ensure-tenant` | Create tenant Namespace via kubectl if missing |
| `gke-fixtures` | Seed GKE fixtures (test-data + public Region URLs) |
| `smoke` | Healthz + list regions + list workspaces |
| `create-workspace` | PUT/GET/DELETE one workspace |
| `stepwise` | Same reads as smoke; 1 VU warmup then 6×10s @ 2..12 VUs; per-op duration SLOs |
| `stress` | Same reads; 1 VU warmup then 12×5s @ 30..360 VUs; per-op duration SLOs |
| `throughput` | Constant-arrival-rate sustained-rate check; `dropped_iterations` + duration SLOs |
| `tf-stress` | Terraform-like stack ×10 workspaces (~5 min, fixed polls, destroy) |
| `tf-net-storage` | Slow workspace bootstrap, then network + block-storage only |
| `ensure-dev-clusters` | Bootstrap local `global`/`regional` KIND dev clusters if missing |
| `*-dev` | `-dev` suffix on any journey target above runs it against the dev clusters (see previous section) |
| `help` | List targets |

## Journeys

### smoke

Cheap gate (1 VU, 1 iteration via `options/smoke.json`):

1. `GET {BASE_URL_GLOBAL}/healthz` and `GET {BASE_URL_REGIONAL}/healthz` → 200 (no auth)
2. `GET …/providers/seca.region/v1/regions` → 200, non-empty `items` (admin bearer)
3. `GET …/providers/seca.workspace/v1/tenants/{tenant}/workspaces` → 200, `items` array

On 403/404 for workspaces, prints a hint for `ensure-tenant` and `deploy-test-data`.

### stepwise

Same light reads as **smoke**, paced at ~1 iteration/s per VU, with a stepped
client ramp (`options/stepwise.json`):

| Phase | Duration | VUs | ~aggregate rate |
|-------|----------|-----|-----------------|
| warmup | 10s | 1 | ~1/s |
| 1 | 10s | 2 | ~2/s |
| 2 | 10s | 4 | ~4/s |
| 3 | 10s | 6 | ~6/s |
| 4 | 10s | 8 | ~8/s |
| 5 | 10s | 10 | ~10/s |
| 6 | 10s | 12 | ~12/s |

Total wall time ≈ 70s (+ graceful ramp-down). Thresholds allow a small check
failure rate under load (`checks>95%`, `http_req_failed<1%`), plus
per-operation `http_req_duration` SLOs (`options/stepwise.json`) so a slow but
*successful* response still fails the run — each request is tagged
(`global_healthz`, `regional_healthz`, `list_regions`, `list_workspaces`) so
thresholds and the summary breakdown are per-operation, not just aggregate.

```bash
make -C test/load stepwise
# or: make -C test load-stepwise
# STEPWISE_PACE_S=1  (default; seconds per VU iteration)
```

### stress

Same light reads as **smoke** / **stepwise**, paced at ~1 iteration/s per VU,
with a steep client ramp (`options/stress.json`):

| Phase | Duration | VUs | ~aggregate rate |
|-------|----------|-----|-----------------|
| warmup | 5s | 1 | ~1/s |
| 1–12 | 5s each | 30, 60, … 360 (+30) | ~30–360/s |

Total wall time ≈ 65s (+ graceful ramp-down). Thresholds are looser than
stepwise under load (`checks>90%`, `http_req_failed<5%`), plus looser
per-operation `http_req_duration` SLOs than stepwise (`options/stress.json`) —
same tagging (`global_healthz`, `regional_healthz`, `list_regions`,
`list_workspaces`). At 360 VUs a single-node dev/KIND control-plane's own API
server can legitimately breach these before the gateway does — that's a
capacity limit of the test environment, not necessarily a gateway bug.

```bash
make -C test/load stress
# or: make -C test load-stress
# STRESS_PACE_S=1  (default; seconds per VU iteration)
```

### throughput

**stepwise/stress ramp *VUs*, not request rate.** Each VU paces itself to
~1 iteration/s; under rising latency, an iteration just takes longer and the
*achieved* rate quietly drops below what the VU count implies — with only
checks/`http_req_failed` gating pass/fail, that shortfall goes unnoticed.

`throughput` fixes the **iteration rate** directly via k6's
`constant-arrival-rate` executor: k6 scales VUs (up to `maxVUs`) to try to
sustain the target rate, and any iteration it can't start in time counts
toward `dropped_iterations` — the actual "we didn't sustain the rate" signal,
gated by a threshold. Same light reads and per-operation tags as
stepwise/stress.

```bash
make -C test/load throughput
# THROUGHPUT_RATE=50            (default; iterations/s)
# THROUGHPUT_DURATION_S=60      (default)
# THROUGHPUT_PRE_VUS=<rate>     (default; preAllocatedVUs)
# THROUGHPUT_MAX_VUS=<rate*4>   (default; maxVUs — headroom for latency spikes)
```

`preAllocatedVUs`/`maxVUs` must comfortably exceed `rate × p95-iteration-latency-in-seconds`,
or the executor itself can't keep up and `dropped_iterations` rises regardless
of API behavior — widen `THROUGHPUT_MAX_VUS` before concluding the API is slow.

### tf-stress

Simulates **10 concurrent Terraform applies** (one workspace each under
`E2E_TENANT`):

| Per user (base + variation) | |
|-----------------------------|--|
| Workspace | 1 (`tf-ws-01` … `tf-ws-10`) |
| Network + route table | 1 each |
| Subnets | 3–5 |
| Block storages | 1–3 |
| Instances | 15–25 |
| NICs | 1 per instance |

Flow: **PUT create** (dependency order) → **fixed_polls** parallel `http.batch`
GETs for ~5 minutes (`TF_JOURNEY_S=300`, minus destroy budget) → **always
DELETE** reverse order.

Concurrency: each poll batch GETs the full stack (~40+ URLs per VU). Ten VUs
run together → well over **10 concurrent** API requests.

```bash
make -C test/load tf-stress
# dashboard: reports/tf-stress-dashboard.html
# TF_JOURNEY_S=300 TF_DESTROY_BUDGET_S=45 TF_ZONE=itbg-1
```

Needs regional fixtures (SKUs / tenant). Dummy delegator: resources stay
`pending`; journey still issues create/poll/delete load.

Writes retry on 429/5xx (`TF_WRITE_RETRIES`, default 8) for admission throttle.
If the GKE `api-throttle` webhook is still tight, open it for the run or
accept higher `http_req_failed` (retries count as failed samples until success).

### tf-net-storage

Lighter TF-like journey that **avoids 404 races**:

1. **`setup()`** creates workspaces **slowly and serially** with a **unique
   `runId` in each name** (`tfns-<runId>-w01`…), so a prior destroy cannot leave
   K8s namespaces in `Terminating` for the same names. Each workspace must
   **PUT + GET 200** before the next; setup **aborts** if any fail — but first
   **deletes whatever workspaces it did manage to create** (`tfDeleteWorkspaces`
   in `lib/tf/apply.js`), so a failed bootstrap doesn't leak workspaces that
   would otherwise affect (or collide with) later runs.
2. **10 VUs** create only **1 network + 1–3 block storages**. Network create
   **retries** while the workspace child namespace is missing or Terminating
   (`TF_NS_READY_ATTEMPTS` / `TF_NS_READY_PAUSE_S`). Polls only created resources.
3. Fixed polls for ~5 min, then **always destroy** (volumes → network → workspace).

```bash
make -C test/load tf-net-storage
# dashboard: reports/tf-net-storage-dashboard.html
# TF_WORKSPACE_CREATE_PAUSE_S=5 TF_JOURNEY_S=300 TF_RUN_ID=myrun
```

If you still see `namespace … being terminated` for **old** fixed names from an
earlier version of this journey, wait for those namespaces to finish deleting
or remove finalizers; new runs use unique names automatically.

### create_workspace

Functional create/get/delete on the regional workspace API:

1. `PUT …/v1/tenants/{tenant}/workspaces/{name}` body `{}` → 200  
   Unique name: `k6-ws-{vu}-{iter}-{timestamp}`
2. `GET` same path → 200, `metadata.name` matches
3. `DELETE` in `finally` → 202 (or 404 if already gone)

Optional Active wait (delegator must be running):

```bash
WAIT_ACTIVE=1 make -C test/load create-workspace
# ACTIVE_TIMEOUT_S=90 ACTIVE_POLL_S=2  (defaults 60s / 2s)
```

## Tenant namespace (`ensure-tenant`)

A tenant is **not** a SECA REST resource. The gateway stores tenant-scoped CRs
in a Kubernetes Namespace:

| Piece | Value |
|-------|--------|
| Name | `hex(sha3-224(tenant))` (same as `deploy.sh` / test-data) |
| Label | `secapi.cloud/tenant-id=<tenant>` |
| Default tenant | `test-tenant` → `f7ec6f666803cd9d9814d4c055217581afbff53f3c35fd6c5e6b444d` |

```bash
make -C test/load ensure-tenant
CREATE_IF_MISSING=0 make -C test/load ensure-tenant   # fail only + manual recipe
```

Behaviour:

1. Namespace exists with the correct label → success (no-op).
2. Missing + `CREATE_IF_MISSING=1` (default) → create + label.
3. Missing + `CREATE_IF_MISSING=0` → non-zero exit + instructions.
4. Exists with wrong or missing label → fail; do not clobber.

**`ensure-tenant` only creates an empty Namespace.** Roles, role-assignments,
SKUs and regions come from **test-data** (included in `kind-stack`). Journeys
that hit RBAC-governed providers need those fixtures, not only the namespace.

## HTML reports

**Graphs are on by default** for every journey (`REPORT_DASHBOARD=1`). After a long
enough run, open:

```bash
make -C test/load stepwise   # or stress
open test/load/reports/k6-dashboard.html
```

| Flag | Default | What you get | Graphs? |
|------|---------|--------------|---------|
| `REPORT_DASHBOARD` | **`1`** | Built-in k6 web dashboard HTML export | **Yes** — latency, VUs, RPS over time |
| `REPORT_HTML` | `0` | k6-reporter via `handleSummary` — tables, checks, thresholds | **No** time-series |

k6 only includes graphs if **test duration > 3 × `K6_WEB_DASHBOARD_PERIOD`**
(default period `10s` → need **>~30s**). **stepwise** / **stress** (~70s) work;
short **smoke** may skip export with a warning.

```bash
# disable graphs
REPORT_DASHBOARD=0 make -C test/load stepwise

# also write table/checks report (opt-in)
REPORT_HTML=1 make -C test/load stress
open test/load/reports/k6-report.html
```

| Env | Default | Meaning |
|-----|---------|---------|
| `REPORT_DASHBOARD` | **`1`** | Web dashboard HTML with **graphs** |
| `K6_DASHBOARD_REPORT` | `reports/k6-dashboard.html` | Path for dashboard export |
| `K6_WEB_DASHBOARD_PERIOD` | `10s` | Aggregation period (shorter → more points; still need duration > 3×) |
| `REPORT_HTML` | `0` | k6-reporter HTML (tables/checks); needs network once for the bundle |
| `K6_HTML_REPORT` | `reports/k6-report.html` | Path for k6-reporter output |
| `K6_REPORT_TITLE` | `ECP k6 report` | k6-reporter title |

Generated files under `reports/` are gitignored. Docker mounts `test/load`
read-write when reports write files.

## Shared library

| Module | Role |
|--------|------|
| `lib/config.js` | Env → base URLs, tenant, auth flags, provider/path helpers |
| `lib/auth.js` | `mintToken()` / `authHeaders()` for dummy auth |
| `lib/http.js` | `get` / `put` / `del` with auth headers + error logging |
| `lib/checks.js` | Status check helpers for journeys |
| `lib/summary.js` | Optional HTML report via `handleSummary` + k6-reporter |
| `lib/status.js` | Count HTTP status codes / classes (`2xx`…`5xx`) for end summary |
| `options/smoke.json` | 1 VU, 1 iteration; fail on check or request errors |

### HTTP status breakdown

Built-in `http_req_failed` is only a pass/fail rate. Journeys that use `lib/http.js`
or `lib/tf/*` also record custom counters. At the end of a run, `handleSummary`
prints:

```text
HTTP status breakdown (custom counters):
  2xx: 1200
  4xx: 40
  5xx: 12
  by code:
    200: 1180
    202: 20
    403: 5
    404: 35
    500: 12
```

You still get per-request lines for unexpected codes via `logIfUnexpected`
(`status=404 expected=200 body=…`).

Dummy token format matches `test/internal/authhelper` (base64 JSON
username/password). JWT is not implemented; extend `mintToken()` when needed.

## Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `BASE_URL_GLOBAL` | _(required for journeys)_ | Global gateway base URL |
| `BASE_URL_REGIONAL` | _(required for journeys)_ | Regional gateway base URL |
| `E2E_TENANT` | `test-tenant` | Tenant id (namespace hashed from this) |
| `AUTH_USER` | `admin` | Dummy authenticator username |
| `AUTH_PASS` | `e2e-admin-pass` | Dummy authenticator password |
| `E2E_AUTH_ENABLED` | `true` | Set `false` to omit `Authorization` |
| `SYSTEM_NAMESPACE` | `e2e-ecp` | Gateway namespace (kubectl helpers) |
| `CREATE_IF_MISSING` | `1` | Create tenant Namespace when missing (`0` = fail only) |
| `WAIT_ACTIVE` | `0` | `1` = poll workspace until `status.state` is Active |
| `ACTIVE_TIMEOUT_S` | `60` | Active poll timeout (seconds) |
| `ACTIVE_POLL_S` | `2` | Active poll interval (seconds) |
| `THROUGHPUT_RATE` | `50` | `throughput`: target iterations/s (constant-arrival-rate) |
| `THROUGHPUT_DURATION_S` | `60` | `throughput`: scenario duration (seconds) |
| `THROUGHPUT_PRE_VUS` | `<rate>` | `throughput`: `preAllocatedVUs` |
| `THROUGHPUT_MAX_VUS` | `<rate>×4` | `throughput`: `maxVUs` (headroom under latency) |
| `K6_IMAGE` | `grafana/k6:0.57.0` | Docker image when native k6 is missing |
| `K6_MIN_VERSION` | `0.54.0` | Minimum accepted native k6 version |
| `KUBECONFIG` | ambient | Cluster used by `ensure-tenant` |

## How k6 is resolved

`make k6-start` (`scripts/k6-start.sh`):

1. Use **native** `k6` on `PATH` if version ≥ `K6_MIN_VERSION`.
2. Else pull and use **Docker** image `K6_IMAGE`.
3. Write `scripts/.k6-cmd` (gitignored) so `run-k6.sh` reuses the choice.

Re-run `k6-start` after installing a better runner.

## Layout

```text
test/load/
  Makefile
  README.md
  scripts/        # k6-start, run-k6, ensure-tenant
  lib/            # config, auth, http, checks, tf/ (apply, urls, bodies, plan)
  journeys/       # hello, _selfcheck, smoke, create_workspace, stepwise,
                  # stress, throughput, tf_stress, tf_net_storage
  options/        # smoke.json, stepwise.json, stress.json, tf-*.json
  observability/  # Grafana dashboards for load-test Prometheus

gateway/scripts/  # dev-url.sh, ensure-dev-clusters.sh
                  # — the *-dev targets' cluster/URL discovery, outside test/load
test/conformance/ionos/scripts/setup-clusters.sh
                  # — the dev clusters themselves (called by ensure-dev-clusters.sh)
```

## Grafana dashboards

Two dashboards ship under `observability/dashboards/`:

| Dashboard | Use during load |
|-----------|-----------------|
| **ECP Gateway** | Auth latency (middleware / authz / RBAC) |
| **ECP Gateway Load Test** | Request counts, memory, GC, process, upstream K8s |

Apply (or refresh) into the load-test monitoring stack:

```bash
make -C test/load apply-dashboards
```

Details: [`observability/README.md`](observability/README.md).

## CI

`smoke` runs on every PR as part of the `chart-smoke` job
(`.github/workflows/pre-merge.yaml` → `ci/scripts/chart-smoke.sh`): after
installing both charts on a throwaway KIND cluster and probing auth, the
script seeds `test/internal/deploy/test-data` (tenant `test-tenant`, RBAC,
region `itbg-bergamo` — matching `test/load`'s own defaults), port-forwards
both gateways, and runs `make -C test/load smoke` against them. `K6_IMAGE` is
not pinned there today — the runner's Docker fallback pulls
`grafana/k6:0.57.0` (this repo's default) fresh each run.

The heavier journeys (`stepwise`, `stress`, `throughput`, `tf-stress`,
`tf-net-storage`) are **not** wired into CI — they run for minutes and their
duration/rate thresholds can fail from the *test environment's* capacity
(e.g. a single-node KIND control-plane under 360 VUs) rather than a real
regression, which is a bad fit for blocking every PR. Follow-up: a
scheduled/manually-triggered workflow for those, separate from `chart-smoke`.

## Out of scope (later)

- JWT auth path
- More resource journeys (network, block storage, …)
- Replacing Go `TestBench` / `benchreport`
