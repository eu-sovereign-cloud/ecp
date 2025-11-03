## Overview
This repository contains the control plane components for the SECA API specification, including global and regional API servers. 
It is designed to manage resources across multiple regions and provide a unified interface for interacting with the SECA ecosystem.

## Prerequisites

- Go 1.24.5 or later
- Docker

## Project Structure
```
.
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
│   │   ├── common           \ - Generic functions
│   │   ├── globalprovider   \ - Global provider logic
│   │   └── regionalprovider \- Regional provider logic
│   └── validation
│   │   ├── filter           \ - Filter labels
├── scripts             \- Utility scripts
```

## Building
See `make help` for a list of build targets.

### Setup Local Development Environment

### Create development kind clusters for global and regional control planes
Note: also builds the docker images for the control plane components.
```bash
make create-dev-clusters
```

### Generate k8s compatible models and crds from go-sdk and manually defined types
```bash
make generate-all
```

### Build docker images
```bash
make docker-build-images
```