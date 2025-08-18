# Control plane

API Servers and control plane for SECA.

## Overview


## Prerequisites

- Go 1.24.5 or later
- Docker

## Project Structure
```
.
├── apis                \- Kubernetes API types and generated CRDs
│   ├── generated       \- Generated CRDs
│   └── regions
│       └── v1          \- API types and generated methods for regions
├── build               \- Dockerfiles for building images
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
### Setup Local Development Environment

### Create development kind clusters for global and regional control planes
```bash
make setup-dev-clusters
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