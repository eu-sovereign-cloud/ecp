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
	"k8s.io/client-go/rest"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/testutil"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/adapters/kubernetes2domain"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/storage"
	imagev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/storage/images/v1"

	model "github.com/eu-sovereign-cloud/ecp/foundation/models"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/adapters/kubernetes"
)

func TestStorageController_CreateAndGetImage(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, storage.AddToScheme(scheme))

	// Use a config copy with higher rate limits to avoid rate limiter exhaustion
	// during the adapter's status polling loop.
	testCfg := rest.CopyConfig(cfg)
	testCfg.QPS = 50
	testCfg.Burst = 100
	dynClient, err := dynamic.NewForConfig(testCfg)
	require.NoError(t, err)

	// Create valid Kubernetes namespace name (lowercase, alphanumeric and hyphens only).
	// Image is tenant-scoped, so the namespace derives from the tenant only.
	tenant := "t-img-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	namespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: tenant})
	const imageName = "test-image"
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
	createController := CreateImage{
		Logger: slog.Default(),
		ImageRepo: kubernetes.NewWriterAdapter(
			dynClient,
			imagev1.ImageGVR,
			slog.Default(),
			kubernetes2domain.MapImageDomainToCR,
			kubernetes2domain.MapCRToImageDomain,
		),
	}

	getController := GetImage{
		Logger: slog.Default(),
		ImageRepo: kubernetes.NewReaderAdapter(
			dynClient,
			imagev1.ImageGVR,
			slog.Default(),
			kubernetes2domain.MapCRToImageDomain,
		),
	}

	updateController := UpdateImage{
		Logger: slog.Default(),
		ImageRepo: kubernetes.NewWriterAdapter(
			dynClient,
			imagev1.ImageGVR,
			slog.Default(),
			kubernetes2domain.MapImageDomainToCR,
			kubernetes2domain.MapCRToImageDomain,
		),
	}

	deleteController := DeleteImage{
		Logger: slog.Default(),
		ImageRepo: kubernetes.NewWriterAdapter(
			dynClient,
			imagev1.ImageGVR,
			slog.Default(),
			kubernetes2domain.MapImageDomainToCR,
			kubernetes2domain.MapCRToImageDomain,
		),
	}

	listController := ListImages{
		Logger: slog.Default(),
		ImageRepo: kubernetes.NewReaderAdapter(
			dynClient,
			imagev1.ImageGVR,
			slog.Default(),
			kubernetes2domain.MapCRToImageDomain,
		),
	}

	t.Run("create_update_list_delete_image", func(t *testing.T) {
		// Create an image domain object
		createDomain := &regional.ImageDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					Name: imageName,
				},
				Scope: scope.Scope{
					Tenant: tenant,
				},
				Labels: map[string]string{
					TenantLabelKey: tenant,
				},
			},
			Spec: regional.ImageSpecDomain{
				BlockStorageRef: regional.ReferenceDomain{
					Resource: "block-storages/source-bs",
				},
				CpuArchitecture: "amd64",
				Initializer:     "none",
				Boot:            "UEFI",
			},
		}

		// Simulate a controller that sets status.state after the CR is created.
		// WriterAdapter.Create polls for status.state to be non-empty; without
		// this, the poll times out because envtest has no real controller.
		statusCfg := rest.CopyConfig(cfg)
		statusCfg.QPS = 50
		statusCfg.Burst = 100
		statusClient, err := dynamic.NewForConfig(statusCfg)
		require.NoError(t, err)

		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, imagev1.ImageGVR, namespace, imageName, map[string]interface{}{
			"sizeMB": int64(1024),
		})

		// Create the image
		createdDomain, err := createController.Do(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, createdDomain)
		require.Equal(t, imageName, createdDomain.Name)
		require.Equal(t, "amd64", createdDomain.Spec.CpuArchitecture)
		require.Equal(t, "block-storages/source-bs", createdDomain.Spec.BlockStorageRef.Resource)

		// Get the image and verify it matches
		metadata := regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: imageName,
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
		}
		retrievedDomain, err := getController.Do(ctx, &metadata)
		require.NoError(t, err)
		require.NotNil(t, retrievedDomain)
		require.Equal(t, imageName, retrievedDomain.Name)
		require.Equal(t, "amd64", retrievedDomain.Spec.CpuArchitecture)
		require.Equal(t, "block-storages/source-bs", retrievedDomain.Spec.BlockStorageRef.Resource)

		// Update the image (change boot type)
		createDomain.Spec.Boot = "BIOS"
		updatedDomain, err := updateController.Do(ctx, createDomain)
		require.NoError(t, err)
		require.Equal(t, "BIOS", updatedDomain.Spec.Boot)

		// Verify update with Get
		retrievedDomain, err = getController.Do(ctx, &metadata)
		require.NoError(t, err)
		require.Equal(t, "BIOS", retrievedDomain.Spec.Boot)

		// List images and verify ours exists
		listParams := model.ListParams{Scope: scope.Scope{Tenant: tenant}}
		items, _, err := listController.Do(ctx, listParams)
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

		// Delete the image (DeleteImage expects IdentifiableResource)
		err = deleteController.Do(ctx, &metadata)
		require.NoError(t, err)
	})

	t.Run("get_nonexistent_image", func(t *testing.T) {
		metadata := regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "missing-image",
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
		}
		_, err := getController.Do(ctx, &metadata)
		require.Error(t, err)
	})
}
