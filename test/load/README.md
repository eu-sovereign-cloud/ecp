# k6 functional API load tests

Black-box **API journeys** run with [k6](https://k6.io/) against a **deployed**
ECP stack (global + regional gateways). Each journey is a short user-shaped
path through the SECA HTTP API (auth headers, tenant paths, create/get/delete).

| This harness | Not this harness |
|--------------|------------------|
| HTTP only, outside the cluster process | Go unit / integration / e2e suites |
| Journey scripts under `journeys/` | Component isolation or CR reconciliation asserts |
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

## Make targets

| Target | What it does |
|--------|----------------|
| `k6-start` | Resolve native k6 or Docker image; write `scripts/.k6-cmd` |
| `k6-version` | Print resolved k6 version |
| `hello` | Print config (no network, URLs optional) |
| `selfcheck` | Offline config/auth checks + `options/smoke.json` |
| `ensure-tenant` | Create tenant Namespace via kubectl if missing |
| `smoke` | Healthz + list regions + list workspaces |
| `create-workspace` | PUT/GET/DELETE one workspace |
| `help` | List targets |

## Journeys

### smoke

Cheap gate (1 VU, 1 iteration via `options/smoke.json`):

1. `GET {BASE_URL_GLOBAL}/healthz` and `GET {BASE_URL_REGIONAL}/healthz` → 200 (no auth)
2. `GET …/providers/seca.region/v1/regions` → 200, non-empty `items` (admin bearer)
3. `GET …/providers/seca.workspace/v1/tenants/{tenant}/workspaces` → 200, `items` array

On 403/404 for workspaces, prints a hint for `ensure-tenant` and `deploy-test-data`.

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

## Shared library

| Module | Role |
|--------|------|
| `lib/config.js` | Env → base URLs, tenant, auth flags, provider/path helpers |
| `lib/auth.js` | `mintToken()` / `authHeaders()` for dummy auth |
| `lib/http.js` | `get` / `put` / `del` with auth headers + error logging |
| `lib/checks.js` | Status check helpers for journeys |
| `options/smoke.json` | 1 VU, 1 iteration; fail on check or request errors |

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
  scripts/     # k6-start, run-k6, ensure-tenant
  lib/         # config, auth, http, checks
  journeys/    # hello, _selfcheck, smoke, create_workspace
  options/     # smoke.json
```

## CI (follow-up)

No CI job is wired yet. A future job needs:

- the same deployed stack as e2e (or a long-lived environment),
- gateway reachability (in-cluster k6 Job, or port-forward/sidecar in the runner),
- `BASE_URL_*` + dummy (or JWT) credentials,
- `make -C test/load smoke` (and optionally `create-workspace`) as a gate.

Prefer pinning `K6_IMAGE` in CI for reproducible runners.

## Out of scope (later)

- Multi-VU load/stress profiles and dashboards
- JWT auth path
- More resource journeys (network, block storage, …)
- Auto port-forward helper
- Replacing Go `TestBench` / `benchreport`
