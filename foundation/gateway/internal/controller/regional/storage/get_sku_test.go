package storage

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	storage "github.com/eu-sovereign-cloud/ecp/foundation/api/block-storage"
	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/block-storage/skus/v1"
	generatedv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

func TestStorageController_GetSKU(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, storage.AddToScheme(scheme))

	dynClient, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)

	// Create valid Kubernetes namespace name (lowercase, alphanumeric and hyphens only)
	tenant := "tenant-get-sku-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	const skuID = "only"
	namespaceGVR := k8sschema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	// Create the namespace object
	namespaceObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": tenant,
			},
		},
	}

	ctx := context.Background()
	_, err = dynClient.Resource(namespaceGVR).Create(ctx, namespaceObj, metav1.CreateOptions{})
	require.NoError(t, err)

	// Cleanup namespace and all resources within it
	t.Cleanup(func() {
		_ = dynClient.Resource(namespaceGVR).Delete(context.Background(), tenant, metav1.DeleteOptions{})
	})

	u := toUnstructured(t, scheme, newStorageSKUCR(skuID, tenant, map[string]string{TenantLabelKey: tenant, "tier": "prod"}, 7500, 10, string(generatedv1.StorageSkuTypeRemoteDurable), false))

	_, err = dynClient.Resource(skuv1.StorageSKUGVR).Namespace(u.GetNamespace()).Create(ctx, u, metav1.CreateOptions{})
	require.NoError(t, err)

	storageSKUAdapter := kubernetes.NewAdapter(
		dynClient,
		skuv1.StorageSKUGVR,
		slog.Default(),
		kubernetes.MapCRToStorageSKUDomain,
	)
	sc := GetSKU{
		Logger:  slog.Default(),
		SKURepo: storageSKUAdapter,
	}
	t.Run("get_existing", func(t *testing.T) {
		metadata := model.Metadata{Namespace: tenant, Name: skuID}
		sku, err := sc.Do(ctx, &metadata)
		require.NoError(t, err)
		require.NotNil(t, sku)
		require.Equal(t, skuID, sku.Name)
		require.NotNil(t, sku.Spec)
		require.Equal(t, int64(7500), sku.Spec.Iops)
	})

	t.Run("get_nonexistent", func(t *testing.T) {
		metadata := model.Metadata{Namespace: tenant, Name: "missing"}
		_, err := sc.Do(ctx, &metadata)
		require.Error(t, err)
	})
}
