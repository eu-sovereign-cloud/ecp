# ECP e2e Plugin

This directory contains an end-to-end testing suite for ECP (Euro-Cloud-Platform) components. It provides a reference implementation and a comprehensive testing environment for ECP plugins and other components like gateways.

By running this e2e implementation, you can observe the entire lifecycle of custom resources (`BlockStorage`, `Role`, `RoleAssignment`, `Workspace`, etc.) as they are processed by the controller. The included `delegator` with its dummy plugins logs the actions it performs (like `Create`, `Delete`) without interacting with a real cloud provider, making it an excellent tool for understanding the resource handling flow.

## Directory Content

-   `cmd/`: Contains the main application entrypoints for the various components (e.g., `delegator`).
-   `pkg/`: Contains the dummy plugin implementations, which log actions but perform no real operations.
-   `build/`: Contains the `Dockerfile` for each component (e.g., `delegator`, `gateway-global`).
-   `deploy/`: Contains Kubernetes manifests (Kustomize) for deploying each component and its necessary RBAC roles.
-   `scripts/`: Provides a suite of helper scripts for building, deploying, testing, and managing the environment, all orchestrated by the `Makefile`.
-   `test/`: Includes integration tests that run against a live Kubernetes cluster to verify the end-to-end flow.

## Getting Started

### Prerequisites

-   [Docker](https://docs.docker.com/get-docker/) (or a compatible container runtime like Podman)
-   [KIND](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
-   [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
-   [make](https://www.gnu.org/software/make/)

## Configuration for Remote Clusters

The `context/` directory (ignored by git) is used to configure the scripts for use with a remote (non-KIND) Kubernetes cluster and container registry.

-   **`context/kubeconfig.yaml`**: If this file is present, `make` recipes like `deploy-all`, `clean-all`, and `test-delegator` will target the cluster defined in this file instead of the default or KIND cluster.

-   **`context/config.env`**: This file can be created to provide credentials for a remote container registry. It is used by the `make push-all` command. It should contain shell variable exports:
    ```shell
    export REGISTRY_URL="my.registry.com"
    export REGISTRY_PROJECT="my-project"
    export REGISTRY_USER="my-user"
    export REGISTRY_PASSWORD="my-password"
    ```

## Local Development & Testing with KIND

The `Makefile` provides a set of powerful and flexible recipes to manage the entire development and testing lifecycle using a local KIND cluster.

### Automated End-to-End Testing (Recommended)

For most development, the `kind-test-delegator` recipe is all you need. It performs the entire test lifecycle in a single command:

```shell
make kind-test-delegator
```

This command will automatically:
1.  Create a new KIND cluster named `e2e-cluster`.
2.  Build the `delegator` container image.
3.  Load the image into the KIND cluster.
4.  Apply the necessary CRDs and deploy the `delegator` manager to the `e2e-ecp` namespace.
5.  Run the Go integration tests against the `delegator`.
6.  Tear down and delete the KIND cluster after the tests complete.

### Manual Lifecycle Management

For debugging or more advanced scenarios, you can use the granular `make` recipes to control each step of the process.

1.  **Start the Cluster:**
    ```shell
    make kind-start
    ```

2.  **Build All Component Images:** The build script now creates tags for both remote and local/KIND registries automatically.
    ```shell
    make build-all
    ```

3.  **Load Images into KIND:**
    ```shell
    make kind-load-all
    ```

4.  **Deploy Components to KIND:**
    ```shell
    make kind-deploy-all
    ```

### Monitoring Resource Handling

Once the components are running, you can watch the logs to see the resource handling cycles in real-time.

To stream the logs for the delegator, run the following command in a separate terminal:
```shell
kubectl logs -f -n e2e-ecp deploy/delegator-depl -c manager
```

### Running Tests Manually

If you have a running cluster with the components deployed, you can run the tests directly:

-   **Against a KIND cluster:**
    ```shell
    make kind-test-delegator
    ```
-   **Against an external cluster:** (Requires `context/kubeconfig.yaml` to be present and configured)
    ```shell
    make test-delegator
    ```

### Cleaning Up

To clean up resources from the KIND cluster without destroying the cluster itself:

```shell
make kind-clean-all
```
This will remove all deployments, services, and CRDs.

To destroy the KIND cluster completely:

```shell
make kind-stop
```

## Authentication & Authorization in e2e

The gateway deployments ship with the Dummy authenticator and SECA RBAC enabled
by default (the defaults changed from the original auth-disabled baseline).
Auth behaviour is driven by environment variables that are read by the
`start-global.sh` / `start-regional.sh` entry-point scripts at startup.

### Gateway deployment env vars

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_ENABLED` | `true` | Set to `false` to run the gateway without any auth (unauthenticated mode). |
| `AUTHZ_ENABLED` | `true` | Set to `false` for authn-only (auth check but no RBAC). Requires `AUTH_ENABLED=true`. |
| `AUTHZ_IMPL` | `cached` | `cached` uses the informer-backed checker (zero K8s round-trips on hot path); `direct` uses the per-request reader (2 K8s List calls per request). |
| `DUMMY_AUTH_USERS` | `/app/users.json` | Path inside the container to the user→password JSON file (mounted from `e2e-dummy-users` ConfigMap). |

### Test-side env vars

| Variable | Default | Description |
|----------|---------|-------------|
| `E2E_AUTH_ENABLED` | `true` (implicit) | Set to `false` to skip all auth-specific test assertions; useful when running against a gateway deployed with `AUTH_ENABLED=false`. |
| `E2E_BENCH` | _(unset)_ | Set to `1` to run the `TestBench` load workload (skipped by default). |
| `E2E_BENCH_REQUESTS` | `500` | Number of requests fired by `TestBench`. |

### Test fixtures: subjects, users, and assignments

The files in `deploy/test-data/` define the RBAC state used by the auth tests.
The table below maps the token subject (`username`) used in tests to the
RoleAssignment that covers them and the net access they should receive:

| Subject | Password | RoleAssignment | Roles | Scope | Expected result |
|---------|----------|----------------|-------|-------|-----------------|
| `admin` | `e2e-admin-pass` | `ra-admin` | `e2e-admin` (all providers, all resources) | all | ✅ All operations |
| `alice` | `alice-pass` | `ra-alice-region-viewer` | `e2e-region-viewer` (`seca.region` `v1/regions`) | `test-tenant` | ✅ List regions (tenant-scoped); ❌ cross-provider ops |
| `bob` | `bob-pass` | `ra-bob-scoped` | `e2e-storage-viewer` (`seca.storage` `block-storages`) | `test-tenant` + region `itbg-bergamo` | ✅ List block-storages in that region; ❌ other regions |
| `carol` | `carol-pass` | `ra-multi-subject` | `e2e-workspace-editor` | `test-tenant` | ✅ Workspace CRUD |
| `dave` | `dave-pass` | `ra-multi-subject` | `e2e-workspace-editor` | `test-tenant` | ✅ Workspace CRUD |
| `erin` | `erin-pass` | `ra-wildcard` (via `*`) + `ra-wrong-tenant` | `e2e-region-viewer` via wildcard; `e2e-admin` scoped to `other-tenant` | `*` / `other-tenant` | ✅ List regions (wildcard scope); ❌ admin ops in `test-tenant` |
| `nobody` | `nobody-pass` | _(none)_ | — | — | ❌ All operations (403) |

> ⚠️ The Dummy authenticator performs no signature verification — any caller who
> knows a valid username+password can claim arbitrary roles. These credentials
> must never be used in production.

### Running auth tests

Auth tests are automatically included when running the normal test suite against
a cluster with `AUTH_ENABLED=true` (the default):

```sh
make kind-test-gateway-global
make kind-test-gateway-regional
```

To skip auth assertions (e.g. against an auth-disabled gateway):

```sh
E2E_AUTH_ENABLED=false make kind-test-gateway-global
```

---

## Benchmarking the Auth Middleware

The `TestBench` load test fires authenticated requests to populate the Prometheus
`ecp_gateway_*_duration_seconds` histograms on the deployed gateway. The
`benchreport` tool then scrapes `/metrics`, computes latency statistics, and
writes a markdown report.

### Quick start

```sh
# 1. Deploy with cached checker (the default)
make kind-deploy-gateway-global   # AUTHZ_IMPL=cached by default

# 2. Fire the load workload
make kind-bench                   # E2E_BENCH=1; default 500 requests

# 3. Scrape metrics and generate the report
IMPL_TAG=cached make report       # writes report/REPORT.md

# 4. Redeploy with the direct checker
AUTHZ_IMPL=direct make kind-deploy-gateway-global

# 5. Fire another load workload and save a second snapshot
E2E_BENCH_REQUESTS=500 make kind-bench
IMPL_TAG=direct SNAP_FILE=report/snap-direct.txt make report

# 6. Merge both snapshots into one comparison report
cd test/e2e && go run ./cmd/benchreport \
    --impl=cached --metrics-file=report/snap.txt \
    --impl=direct --metrics-file=report/snap-direct.txt \
    --out=report/REPORT.md
```

### Reading the report

`report/REPORT.md` contains three latency tables — one per histogram — with rows
for each `impl/label` combination and columns:

| Column | Meaning |
|--------|---------|
| `count` | Total number of observations. |
| `avg (ms)` | Arithmetic mean latency in milliseconds. |
| `p50 (ms)` | Median latency (50th percentile), interpolated from buckets. |
| `p90 (ms)` | 90th-percentile latency. |
| `p99 (ms)` | 99th-percentile latency. |

Expected comparison pattern:

- `ecp_gateway_rbac_fetch_duration_seconds{impl="cached"}` p99 should be
  orders of magnitude lower than `impl="direct"` (in-memory read vs K8s List).
- `ecp_gateway_authz_check_duration_seconds` should mirror the fetch delta.
- `ecp_gateway_auth_middleware_duration_seconds` differences reflect the
  checker cost amortised over the full request (provider handler is a constant).

---

## Working with a Remote Cluster

To deploy and test against a remote Kubernetes cluster, ensure you have configured your `context/kubeconfig.yaml` and `context/config.env` files as described above.

1.  **Build All Images:**
    ```shell
    make build-all
    ```

2.  **Push Images to Remote Registry:**
    ```shell
    make push-all
    ```

3.  **Deploy Components to Remote Cluster:**
    ```shell
    make deploy-all
    ```

4.  **Run Tests Against Remote Cluster:**
    ```shell
    make test-delegator
    ```

5.  **Clean Up Remote Cluster:**
    ```shell
    make clean-all
    ```
This will remove all deployments, services, and CRDs that were created.
