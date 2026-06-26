//go:build envtest

package kubernetes_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	testutil "github.com/eu-sovereign-cloud/ecp/resource/common/frontend/testutil"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"

	. "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
)

func TestBlockStorageBackend_CreateAndGetBlockStorage(t *testing.T) {
	t.Parallel()

	// Use a config copy with higher rate limits to avoid rate limiter exhaustion
	// during the adapter's status polling loop.
	testCfg := rest.CopyConfig(cfg)
	testCfg.QPS = 50
	testCfg.Burst = 100
	dynClient, err := dynamic.NewForConfig(testCfg)
	require.NoError(t, err)

	// Create valid Kubernetes namespace name (lowercase, alphanumeric and hyphens only).
	// Keep tenant short to fit within 63 char label limit.
	tenant := "t-bs-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant})
	const blockStorageName = "test-block-storage"
	namespaceGVR := k8sschema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	// Create the namespace so the CRD resources have somewhere to land.
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

	// Cleanup namespace and all resources within it.
	t.Cleanup(func() {
		_ = dynClient.Resource(namespaceGVR).Delete(context.Background(), namespace, metav1.DeleteOptions{})
	})

	// Build writer and reader adapters directly from the kubernetes backend package.
	writerRepo := k8sadapter.NewWriterAdapter[*bsdom.BlockStorage](
		dynClient,
		BlockStorageGVR,
		slog.Default(),
		BlockStorageToCR,
		BlockStorageFromCR,
	)
	readerRepo := k8sadapter.NewReaderAdapter[*bsdom.BlockStorage](
		dynClient,
		BlockStorageGVR,
		slog.Default(),
		BlockStorageFromCR,
	)

	t.Run("create_update_list_delete_block_storage", func(t *testing.T) {
		createDomain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: blockStorageName},
				Scope:          kernelresource.Scope{Tenant: tenant},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: bsdom.BlockStorageSpec{
				SizeGB: 100,
				SkuRef: commondomain.Reference{Resource: "standard-ssd"},
			},
		}

		// Simulate a status controller so the CR's status.state becomes non-empty.
		// WriterAdapter.Create polls for status.state; without this the poll times out
		// because envtest has no real controller.
		statusCfg := rest.CopyConfig(cfg)
		statusCfg.QPS = 50
		statusCfg.Burst = 100
		statusClient, err := dynamic.NewForConfig(statusCfg)
		require.NoError(t, err)

		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, BlockStorageGVR, namespace, blockStorageName, map[string]interface{}{
			"sizeGB": int64(100),
		})

		// Create the block storage.
		resultPtr, err := writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, resultPtr)
		created := *resultPtr
		require.Equal(t, blockStorageName, created.Name)
		require.Equal(t, 100, created.Spec.SizeGB)
		require.Equal(t, "standard-ssd", created.Spec.SkuRef.Resource)

		// Get the block storage and verify it matches.
		getDomain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: blockStorageName},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err = readerRepo.Load(ctx, &getDomain)
		require.NoError(t, err)
		require.NotNil(t, getDomain)
		require.Equal(t, blockStorageName, getDomain.Name)
		require.Equal(t, 100, getDomain.Spec.SizeGB)
		require.Equal(t, "standard-ssd", getDomain.Spec.SkuRef.Resource)

		// Update the block storage spec.
		createDomain.Spec.SizeGB = 200
		createDomain.ResourceVersion = created.ResourceVersion
		updatedPtr, err := writerRepo.Update(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, updatedPtr)
		updated := *updatedPtr
		require.Equal(t, 200, updated.Spec.SizeGB)

		// Verify update with a Get.
		getDomain2 := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: blockStorageName},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err = readerRepo.Load(ctx, &getDomain2)
		require.NoError(t, err)
		require.Equal(t, 200, getDomain2.Spec.SizeGB)

		// List block storages and verify ours exists.
		var items []*bsdom.BlockStorage
		listParams := kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: tenant}}
		_, err = readerRepo.List(ctx, listParams, &items)
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

		// Delete the block storage.
		delDomain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: blockStorageName},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err = writerRepo.Delete(ctx, delDomain)
		require.NoError(t, err)
	})

	t.Run("get_nonexistent_block_storage", func(t *testing.T) {
		getDomain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: "missing-block-storage"},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err := readerRepo.Load(ctx, &getDomain)
		require.Error(t, err)
	})
}
