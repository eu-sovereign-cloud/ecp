# ECP IonOS Delegator Plugin

This directory contains the implementation of a Delegator plugin for the IonOS Cloud Service Provider (CSP). Uses Crossplane to manage actual IonOS cloud resources, such as block storage volumes and virtual machines (workspaces).

## Overview

The IonOS plugin integrates with Crossplane's IonOS provider to provision and manage cloud resources. Creates and manages Crossplane Custom Resources (CRs) that represent IonOS resources.

### Key Components

- **Crossplane Integration**: Uses the [Crossplane IonOS Provider](https://github.com/ionos-cloud/crossplane-provider-ionoscloud) to interact with IonOS APIs via Kubernetes CRDs.
- **Resource Mapping**:
  - `BlockStorage` -> Crossplane `Volume` CRD
  - `Workspace` -> Crossplane `Server` CRD (or appropriate IonOS resource)
- **Lifecycle Management**: Handles create, update (e.g., resize), and delete operations by reconciling Crossplane CRs.

## Directory Content

- `cmd/`: Main application entrypoint for the delegator.
- `pkg/`: IonOS plugin implementation using Crossplane CRDs.
- `build/`: Dockerfile for building the delegator image.
- `deploy/`: Kubernetes manifests (Kustomize) for deploying the delegator, including Crossplane setup.
- `scripts/`: Helper scripts for building, deploying, and managing the cluster.
- `test/`: Integration tests for end-to-end verification.

## Prerequisites

- Docker (or Podman)
- KIND or a Kubernetes cluster
- kubectl
- make
- Crossplane CLI (optional, for manual setup)

## Getting Started

### 1. Set Up Crossplane

Before deploying the plugin, ensure Crossplane is installed in your cluster:

```shell
# Install Crossplane
kubectl create namespace crossplane-system
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm install crossplane crossplane-stable/crossplane --namespace crossplane-system --create-namespace

# Install IonOS Provider
kubectl crossplane install provider crossplane/provider-ionoscloud:v0.5.0
```

Create a ProviderConfig for IonOS with your credentials (replace with actual values):

```yaml
apiVersion: ionoscloud.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: ionos-provider-config
spec:
  credentials:
    source: Secret
    secretRef:
      name: ionos-creds
      namespace: crossplane-system
      key: credentials
---
apiVersion: v1
kind: Secret
metadata:
  name: ionos-creds
  namespace: crossplane-system
type: Opaque
data:
  credentials: <base64-encoded-ionos-credentials>
```

### 2. Deploy the IonOS Delegator

Use the provided Makefile for easy deployment:

```shell
make kind-start
```

This will:
1. Create a KIND cluster.
2. Install Crossplane and the IonOS provider.
3. Build and load the IonOS delegator image.
4. Deploy the delegator with necessary RBAC.

### 3. Monitor Resource Handling

Stream logs to observe Crossplane resource reconciliation:

```shell
kubectl logs -f -n ecp-ionos-delegator deploy/ionos-delegator-depl -c manager
```

### 4. Run Integration Tests

```shell
make test-integration
```

## Implementation Sketch

### Plugin Structure

The plugin implements the same interface as the dummy plugin but uses Crossplane CRDs.

#### BlockStorage Plugin (`pkg/plugin/block_storage.go`)

```go
type BlockStorage struct {
    client client.Client
    logger *slog.Logger
}

func (b *BlockStorage) Create(ctx context.Context, resource *regional.BlockStorageDomain) error {
    // Create Crossplane Volume CRD
    volume := &ionosv1alpha1.Volume{
        ObjectMeta: metav1.ObjectMeta{
            Name:      resource.GetName(),
            Namespace: "crossplane-system", // or appropriate namespace
        },
        Spec: ionosv1alpha1.VolumeSpec{
            ForProvider: ionosv1alpha1.VolumeParameters{
                Name:     resource.Spec.Name,
                Size:     &resource.Spec.SizeGB,
                Type:     "HDD", // map from spec
                // other IonOS-specific params
            },
        },
    }
    return b.client.Create(ctx, volume)
}

// Similar for Delete and IncreaseSize (update spec and reconcile)
```

#### Workspace Plugin (`pkg/plugin/workspace.go`)

Map Workspace to IonOS Server or Datacenters as needed.

### Deployment Considerations

- **RBAC**: The delegator service account needs permissions to create/update/delete Crossplane CRDs in the crossplane-system namespace.
- **Namespaces**: Decide on namespaces for Crossplane resources vs. ECP resources.
- **Error Handling**: Implement retries and status checks based on Crossplane CR status.
- **Secrets Management**: Use Kubernetes secrets for IonOS credentials, referenced in ProviderConfig.

### Differences from Dummy Plugin

- **Real Operations**: Instead of logging, performs actual cloud provisioning via Crossplane.
- **Dependencies**: Adds Crossplane and IonOS provider Go modules.
- **Deployment**: Includes Crossplane installation in deploy manifests.
- **Configuration**: Requires IonOS credentials setup.

## Next Steps

- Implement the full plugin logic with proper error handling and status synchronization.
- Add unit tests for plugin methods.
- Enhance integration tests to verify actual IonOS resource creation (requires real credentials).
- Handle advanced features like snapshots, backups, etc., if supported by the model.</content>
<parameter name="filePath">/home/cguran/surse/seca/ecp/foundation/plugin/ionos/README.md
