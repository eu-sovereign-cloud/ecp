# ECP Dummy Delegator Plugin

This directory contains a template and example implementation of a Delegator with a Dummy Plugin. Its primary purpose is to serve as a reference and a starting point for developing new, fully-functional plugins for the ECP platform.

By running this dummy implementation, you can observe the entire lifecycle of custom resources (`BlockStorage`, `Workspace`) as they are processed by the controller. The dummy plugin logs the actions it performs (like `Create`, `Delete`, `IncreaseSize`) without interacting with a real cloud provider, making it an excellent tool for understanding the resource handling flow.

## Directory Content

-   `cmd/`: Contains the main application entrypoint for the delegator.
-   `pkg/`: Contains the dummy plugin implementation, which logs actions but performs no real operations.
-   `build/`: Contains the `Dockerfile` for building the delegator image.
-   `deploy/`: Contains Kubernetes manifests (Kustomize) for deploying the delegator and its necessary RBAC roles.
-   `scripts/`: Provides helper scripts for building the Docker image, deploying to Kubernetes, and managing a local KIND cluster.
-   `test/`: Includes integration tests that run against a live KIND cluster to verify the end-to-end flow.

## Getting Started

### Prerequisites

-   [Docker](https://docs.docker.com/get-docker/) (or a compatible container runtime like Podman)
-   [KIND](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
-   [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
-   [make](https://www.gnu.org/software/make/)

### Launching the Delegator in KIND

You can start a local KIND cluster and deploy the dummy delegator with a single command:

```shell
make kind-start
```

This command will:
1.  Create a new KIND cluster named `dummy-delegator-cluster`.
2.  Build the `ecp-dummy-delegator` Docker image.
3.  Load the image into the KIND cluster.
4.  Apply the necessary CRDs and deploy the delegator manager to the `ecp-dummy-delegator` namespace.

### Monitoring Resource Handling

Once the delegator is running, you can watch its logs to see the resource handling cycles in real-time.

To stream the logs, run the following command in a separate terminal:

```shell
kubectl logs -f -n ecp-dummy-delegator deploy/dummy-delegator-depl -c manager
```

You will see logs from the controller as it reconciles resources, and messages from the dummy plugin indicating which actions are being called.

### Running Integration Tests

The integration tests create, update, and delete `BlockStorage` and `Workspace` resources in the running KIND cluster and assert that they reach the expected states.

To run the tests, use the following command:

```shell
make test-integration
```

This will execute the Go tests located in the `test/integration/` directory. You can observe the corresponding reconciliation logs in the terminal where you are streaming the pod logs.

### Cleaning Up

To delete the KIND cluster and all deployed resources, run:

```shell
make kind-stop
```
