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
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1"

	. "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1/backend/kubernetes"
)

func TestImageBackend_CreateAndGetImage(t *testing.T) {
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
	tenant := "t-img-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant})
	const imageName = "test-image"
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
	writerRepo := k8sadapter.NewWriterAdapter[*imgdom.Image](
		dynClient,
		ImageGVR,
		slog.Default(),
		ImageToCR,
		ImageFromCR,
	)
	readerRepo := k8sadapter.NewReaderAdapter[*imgdom.Image](
		dynClient,
		ImageGVR,
		slog.Default(),
		ImageFromCR,
	)

	t.Run("create_update_list_delete_image", func(t *testing.T) {
		createDomain := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: imageName},
				Scope:          kernelresource.Scope{Tenant: tenant},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: imgdom.ImageSpec{
				BlockStorageRef: commondomain.Reference{Resource: "block-storages/source-bs"},
				CpuArchitecture: "amd64",
				Boot:            "UEFI",
				Initializer:     "none",
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
		go testutil.SimulateStatusController(statusCtx, statusClient, ImageGVR, namespace, imageName, map[string]interface{}{
			"sizeMB": int64(628),
		})

		// Create the image.
		resultPtr, err := writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, resultPtr)
		created := *resultPtr
		require.Equal(t, imageName, created.Name)
		require.Equal(t, "amd64", created.Spec.CpuArchitecture)
		require.Equal(t, "block-storages/source-bs", created.Spec.BlockStorageRef.Resource)

		// Get the image and verify it matches.
		getDomain := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: imageName},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err = readerRepo.Load(ctx, &getDomain)
		require.NoError(t, err)
		require.NotNil(t, getDomain)
		require.Equal(t, imageName, getDomain.Name)
		require.Equal(t, "amd64", getDomain.Spec.CpuArchitecture)
		require.Equal(t, "block-storages/source-bs", getDomain.Spec.BlockStorageRef.Resource)

		// Update the image spec.
		createDomain.Spec.Boot = "BIOS"
		createDomain.ResourceVersion = created.ResourceVersion
		updatedPtr, err := writerRepo.Update(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, updatedPtr)
		updated := *updatedPtr
		require.Equal(t, "BIOS", updated.Spec.Boot)

		// Verify update with a Get.
		getDomain2 := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: imageName},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err = readerRepo.Load(ctx, &getDomain2)
		require.NoError(t, err)
		require.Equal(t, "BIOS", getDomain2.Spec.Boot)

		// List images and verify ours exists.
		var items []*imgdom.Image
		listParams := kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: tenant}}
		_, err = readerRepo.List(ctx, listParams, &items)
		require.NoError(t, err)
		require.NotEmpty(t, items)
		found := false
		for _, it := range items {
			if it != nil && it.Name == imageName {
				found = true
				break
			}
		}
		require.True(t, found, "expected image to be present in list")

		// Delete the image.
		delDomain := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: imageName},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err = writerRepo.Delete(ctx, delDomain)
		require.NoError(t, err)
	})

	t.Run("get_nonexistent_image", func(t *testing.T) {
		getDomain := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: "missing-image"},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err := readerRepo.Load(ctx, &getDomain)
		require.Error(t, err)
	})
}
