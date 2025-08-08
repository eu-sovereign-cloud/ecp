# Control plane

API Server and control plane for SECA.

## Overview


## Prerequisites

- Go 1.24.5 or later
- kubectl

## Project Structure
```
.
├── api/
│   └── handlers - api server related handlers
├── cmd - cobra command lines for each tool
├── config - kubernetes resources
├── internal
│   └── validation - validation logic for rest requests
├── pkg
│   ├── apiserver - API server for handling requests
│   ├── kubeclient - client for managing Kubernetes resources(xrds, secrets, etc.)
│   ├── logger
│   └── providers - provider related logic
│       └── ...
```
### Setup Local Development Environment

#### Create Kind cluster with Crossplane and IONOS provider
```bash
make crossplane-local-dev
```