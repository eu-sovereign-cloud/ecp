## Overview
This repository contains the control plane components for the SECA API specification, including global and regional API servers. 
It is designed to manage resources across multiple regions and provide a unified interface for interacting with the SECA ecosystem.

## Prerequisites

- Go 1.24.5 or later
- Docker

## Project Structure
```
.
├── apis                \- Kubernetes API types and generated CRDs
│   ├── generated       \- Generated CRDs and types
├── build               \- Dockerfiles for building images
├── examples            \- Examples for crds
├── cmd                 \- Command-line entry points
├── config              \- Kubernetes resource manifests
│   └── setup           \ - Setup manifests for dev clusters
├── internal            \- Internal logic (handlers, clients, providers, validation, logger)
│   ├── handler         \- HTTP handlers for API endpoints
│   ├── kubeclient      \ - Kubernetes client utilities using dynamic client
│   ├── logger
│   ├── provider
│   │   ├── globalprovider \ - Global provider logic
│   │   └── regionalprovider \- Regional provider logic
│   └── validation
├── scripts             \- Utility scripts
├── tools               \- Tool dependencies
```

## Generate Kubernetes code and CRDs with controller-gen:

See following pages to learning how it is done:
- https://kubebuilder.io/reference/generating-crd.html
- https://kubebuilder.io/reference/markers.html

When you make changes to the auto-generated code, you'll need to regenerate the code and CRDs using the following command:

```bash
make generate-crds
```

## Building

See `make help` for a list of build targets.

### Setup Local Development Environment

### Create development kind clusters for global and regional control planes
Note: also builds the docker images for the control plane components.
```bash
make create-dev-clusters
```

#### Create Kind cluster with Crossplane
```bash
make crossplane-local-dev
```

### Generate crds from Spec
```bash
make generate-crds
```

### Build docker images
```bash
make docker-build-images
```


# Run/Debug API Server locally 
```bash
make setup-dev-clusters
```
Start `globalapiserver` or `regionalapiserver` in debug mode.
Set the environment variable `KUBECONFIG` to point to the kubeconfig file of the kind cluster you want to use.
