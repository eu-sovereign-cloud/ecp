package storage

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/block-storages/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

func TestStorageController_CreateAndGetBlockStorage(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, storage.AddToScheme(scheme))

	dynClient, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)

	// Create valid Kubernetes namespace name (lowercase, alphanumeric and hyphens only)
	// Keep tenant short to fit within 63 char label limit
	tenant := "t-bs-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	namespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: tenant})
	const blockStorageName = "test-block-storage"
	namespaceGVR := k8sschema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	// Create the namespace object
	namespaceObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": namespace,
			},
		},
	}

	ctx := context.Background()
	_, err = dynClient.Resource(namespaceGVR).Create(ctx, namespaceObj, metav1.CreateOptions{})
	require.NoError(t, err)

	// Cleanup namespace and all resources within it
	t.Cleanup(func() {
		_ = dynClient.Resource(namespaceGVR).Delete(context.Background(), namespace, metav1.DeleteOptions{})
	})

	// Setup controllers
	createController := CreateBlockStorage{
		Logger: slog.Default(),
		BlockStorageRepo: kubernetes.NewWriterAdapter(
			dynClient,
			blockstoragev1.BlockStorageGVR,
			slog.Default(),
			kubernetes.MapBlockStorageDomainToCR,
			kubernetes.MapCRToBlockStorageDomain,
		),
	}

	getController := GetBlockStorage{
		Logger: slog.Default(),
		BlockStorageRepo: kubernetes.NewReaderAdapter(
			dynClient,
			blockstoragev1.BlockStorageGVR,
			slog.Default(),
			kubernetes.MapCRToBlockStorageDomain,
		),
	}

	updateController := UpdateBlockStorage{
		Logger: slog.Default(),
		BlockStorageRepo: kubernetes.NewWriterAdapter(
			dynClient,
			blockstoragev1.BlockStorageGVR,
			slog.Default(),
			kubernetes.MapBlockStorageDomainToCR,
			kubernetes.MapCRToBlockStorageDomain,
		),
	}

	deleteController := DeleteBlockStorage{
		Logger: slog.Default(),
		BlockStorageRepo: kubernetes.NewWriterAdapter(
			dynClient,
			blockstoragev1.BlockStorageGVR,
			slog.Default(),
			kubernetes.MapBlockStorageDomainToCR,
			kubernetes.MapCRToBlockStorageDomain,
		),
	}

	listController := ListBlockStorages{
		Logger: slog.Default(),
		BlockStorageRepo: kubernetes.NewReaderAdapter(
			dynClient,
			blockstoragev1.BlockStorageGVR,
			slog.Default(),
			kubernetes.MapCRToBlockStorageDomain,
		),
	}

	t.Run("create_update_list_delete_block_storage", func(t *testing.T) {
		// Create a block storage domain object
		createDomain := &regional.BlockStorageDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					Name: blockStorageName,
				},
				Scope: scope.Scope{
					Tenant: tenant,
				},
				Labels: map[string]string{
					TenantLabelKey: tenant,
				},
			},
			Spec: regional.BlockStorageSpec{
				SizeGB: 100,
				SkuRef: regional.ReferenceObject{
					Resource: "standard-ssd",
				},
			},
		}

		// Create the block storage
		createdDomain, err := createController.Do(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, createdDomain)
		require.Equal(t, blockStorageName, createdDomain.Name)
		require.Equal(t, 100, createdDomain.Spec.SizeGB)
		require.Equal(t, "standard-ssd", createdDomain.Spec.SkuRef.Resource)

		// Get the block storage and verify it matches
		metadata := regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: blockStorageName,
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
		}
		retrievedDomain, err := getController.Do(ctx, &metadata)
		require.NoError(t, err)
		require.NotNil(t, retrievedDomain)
		require.Equal(t, blockStorageName, retrievedDomain.Name)
		require.Equal(t, 100, retrievedDomain.Spec.SizeGB)
		require.Equal(t, "standard-ssd", retrievedDomain.Spec.SkuRef.Resource)

		// Update the block storage
		createDomain.Spec.SizeGB = 200
		updatedDomain, err := updateController.Do(ctx, createDomain)
		require.NoError(t, err)
		require.Equal(t, 200, updatedDomain.Spec.SizeGB)

		// Verify update with Get
		retrievedDomain, err = getController.Do(ctx, &metadata)
		require.NoError(t, err)
		require.Equal(t, 200, retrievedDomain.Spec.SizeGB)

		// List block storages and verify ours exists
		listParams := model.ListParams{Scope: scope.Scope{Tenant: tenant}}
		items, _, err := listController.Do(ctx, listParams)
		require.NoError(t, err)
		require.NotEmpty(t, items)
		found := false
		for _, it := range items {
			if it != nil && it.Name == blockStorageName {
				found = true
				break
			}
		}
		require.True(t, found, "expected block storage to be present in list")

		// Delete the block storage (DeleteBlockStorage expects IdentifiableResource)
		err = deleteController.Do(ctx, &metadata)
		require.NoError(t, err)

		// Verify deletion
		_, err = getController.Do(ctx, &metadata)
		require.Error(t, err)
	})

	t.Run("get_nonexistent_block_storage", func(t *testing.T) {
		metadata := regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "missing-block-storage",
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
		}
		_, err := getController.Do(ctx, &metadata)
		require.Error(t, err)
	})
}
