package kubernetes_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/storage/storage-sku/v1"

	. "github.com/eu-sovereign-cloud/ecp/resource/storage/storage-sku/v1/backend/kubernetes"
)

const (
	testProvider = "seca.storage/v1"
	testRegion   = "eu-central-1"
	testTenant   = "tn-1"
)

// newStorageSKUCR builds a StorageSKU CR with the given identity, internal labels
// and spec. createdAt is used as the creation timestamp when non-zero.
func newStorageSKUCR(name, version string, iops, minVolumeSize int, skuType string, createdAt time.Time) *StorageSKU {
	cr := &StorageSKU{
		TypeMeta: metav1.TypeMeta{Kind: StorageSKUKind, APIVersion: GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			ResourceVersion: version,
			Labels: map[string]string{
				k8slabels.InternalProviderLabel: testProvider,
				k8slabels.InternalRegionLabel:   testRegion,
				k8slabels.InternalTenantLabel:   testTenant,
			},
		},
		Spec: StorageSkuSpec{
			Iops:          iops,
			MinVolumeSize: minVolumeSize,
			Type:          StorageSkuSpecType(skuType),
		},
	}
	if !createdAt.IsZero() {
		cr.SetCreationTimestamp(metav1.Time{Time: createdAt})
	}
	return cr
}

func toUnstructured(t *testing.T, cr *StorageSKU) *unstructured.Unstructured {
	t.Helper()
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cr)
	require.NoError(t, err)
	u := &unstructured.Unstructured{Object: m}
	u.SetGroupVersionKind(StorageSKUGVK)
	return u
}

func TestStorageSKUFromCR(t *testing.T) {
	created := time.Unix(1700000000, 0)

	t.Run("from_concrete_cr", func(t *testing.T) {
		cr := newStorageSKUCR("seca.rd2k", "7", 2000, 50, "remote-durable", created)

		sku, err := StorageSKUFromCR(cr)
		require.NoError(t, err)
		require.NotNil(t, sku)

		require.Equal(t, "seca.rd2k", sku.Name)
		require.Equal(t, "7", sku.ResourceVersion)
		require.Equal(t, int64(2000), sku.Spec.IOPS)
		require.Equal(t, int64(50), sku.Spec.MinVolumeSize)
		require.Equal(t, "remote-durable", sku.Spec.Type)

		// Identity is sourced from the internal labels.
		require.Equal(t, testProvider, sku.Provider)
		require.Equal(t, testRegion, sku.Region)
		require.Equal(t, testTenant, sku.Tenant)

		// Both timestamps derive from the creation timestamp; no deletion.
		require.Equal(t, created.UTC(), sku.CreatedAt.UTC())
		require.Equal(t, created.UTC(), sku.UpdatedAt.UTC())
		require.Nil(t, sku.DeletedAt)
	})

	t.Run("from_unstructured", func(t *testing.T) {
		cr := newStorageSKUCR("seca.le40k", "3", 40000, 1, "local-ephemeral", created)

		sku, err := StorageSKUFromCR(toUnstructured(t, cr))
		require.NoError(t, err)
		require.NotNil(t, sku)

		require.Equal(t, "seca.le40k", sku.Name)
		require.Equal(t, int64(40000), sku.Spec.IOPS)
		require.Equal(t, int64(1), sku.Spec.MinVolumeSize)
		require.Equal(t, "local-ephemeral", sku.Spec.Type)
		require.Equal(t, testTenant, sku.Tenant)
	})

	t.Run("deletion_timestamp_is_propagated", func(t *testing.T) {
		cr := newStorageSKUCR("seca.ld5k", "1", 5000, 50, "local-durable", created)
		deletedAt := metav1.Time{Time: created.Add(time.Hour)}
		cr.SetDeletionTimestamp(&deletedAt)

		sku, err := StorageSKUFromCR(cr)
		require.NoError(t, err)
		require.NotNil(t, sku.DeletedAt)
		require.Equal(t, deletedAt.UTC(), sku.DeletedAt.UTC())
	})

	t.Run("unsupported_type_errors", func(t *testing.T) {
		sku, err := StorageSKUFromCR(&metav1.PartialObjectMetadata{})
		require.Error(t, err)
		require.Nil(t, sku)
	})
}

func TestStorageSKUToCR(t *testing.T) {
	t.Run("populates_spec_and_identity", func(t *testing.T) {
		dom := &skudom.StorageSKU{
			Spec: skudom.StorageSKUSpec{IOPS: 10000, MinVolumeSize: 50, Type: "remote-durable"},
		}
		dom.Name = "seca.rd10k"
		dom.ResourceVersion = "9"

		obj, err := StorageSKUToCR(dom)
		require.NoError(t, err)

		cr, ok := obj.(*StorageSKU)
		require.True(t, ok, "expected *StorageSKU, got %T", obj)
		require.Equal(t, "seca.rd10k", cr.GetName())
		require.Equal(t, "9", cr.GetResourceVersion())
		require.Equal(t, StorageSKUGVK, cr.GroupVersionKind())
		require.Equal(t, 10000, cr.Spec.Iops)
		require.Equal(t, 50, cr.Spec.MinVolumeSize)
		require.Equal(t, StorageSkuSpecType("remote-durable"), cr.Spec.Type)
	})

	t.Run("spec_survives_round_trip", func(t *testing.T) {
		obj, err := StorageSKUToCR(&skudom.StorageSKU{
			Spec: skudom.StorageSKUSpec{IOPS: 500, MinVolumeSize: 20, Type: "local-durable"},
		})
		require.NoError(t, err)

		sku, err := StorageSKUFromCR(obj)
		require.NoError(t, err)
		require.Equal(t, int64(500), sku.Spec.IOPS)
		require.Equal(t, int64(20), sku.Spec.MinVolumeSize)
		require.Equal(t, "local-durable", sku.Spec.Type)
	})

	t.Run("nil_errors", func(t *testing.T) {
		obj, err := StorageSKUToCR(nil)
		require.Error(t, err)
		require.Nil(t, obj)
	})
}
