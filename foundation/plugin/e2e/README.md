# ECP e2e Plugin

This directory contains an end-to-end testing suite for ECP (Euro-Cloud-Platform) components. It provides a reference implementation and a comprehensive testing environment for ECP plugins and other components like gateways.

By running this e2e implementation, you can observe the entire lifecycle of custom resources (`BlockStorage`, `Workspace`, etc.) as they are processed by the controller. The included `delegator` with its dummy plugins logs the actions it performs (like `Create`, `Delete`) without interacting with a real cloud provider, making it an excellent tool for understanding the resource handling flow.

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
