package kubernetes_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku"

	. "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku/backend/kubernetes"
)

const (
	testProvider = "seca.compute/v1"
	// testProviderLabel is testProvider as it is stored on the CR: "/" is not a
	// legal label value character, so it is encoded as "_".
	testProviderLabel = "seca.compute_v1"
	testRegion        = "eu-central-1"
	testTenant        = "tn-1"
)

// newInstanceSKUCR builds an InstanceSKU CR with the given identity, internal labels
// and spec. createdAt is used as the creation timestamp when non-zero.
func newInstanceSKUCR(name, version string, vcpu, ram int, createdAt time.Time) *InstanceSKU {
	cr := &InstanceSKU{
		TypeMeta: metav1.TypeMeta{Kind: InstanceSKUKind, APIVersion: GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			ResourceVersion: version,
			Labels: map[string]string{
				k8slabels.InternalProviderLabel: testProviderLabel,
				k8slabels.InternalRegionLabel:   testRegion,
				k8slabels.InternalTenantLabel:   testTenant,
			},
		},
		Spec: InstanceSkuSpec{
			VCPU: vcpu,
			Ram:  ram,
		},
	}
	if !createdAt.IsZero() {
		cr.SetCreationTimestamp(metav1.Time{Time: createdAt})
	}
	return cr
}

func toUnstructured(t *testing.T, cr *InstanceSKU) *unstructured.Unstructured {
	t.Helper()
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cr)
	require.NoError(t, err)
	u := &unstructured.Unstructured{Object: m}
	u.SetGroupVersionKind(InstanceSKUGVK)
	return u
}

func TestInstanceSKUFromCR(t *testing.T) {
	created := time.Unix(1700000000, 0)

	t.Run("from_concrete_cr", func(t *testing.T) {
		cr := newInstanceSKUCR("compute-sku-1", "7", 4, 8, created)

		sku, err := InstanceSKUFromCR(cr)
		require.NoError(t, err)
		require.NotNil(t, sku)

		require.Equal(t, "compute-sku-1", sku.Name)
		require.Equal(t, "7", sku.ResourceVersion)
		require.Equal(t, 4, sku.Spec.VCPU)
		require.Equal(t, 8, sku.Spec.Ram)

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
		cr := newInstanceSKUCR("compute-sku-2", "3", 2, 4, created)

		sku, err := InstanceSKUFromCR(toUnstructured(t, cr))
		require.NoError(t, err)
		require.NotNil(t, sku)

		require.Equal(t, "compute-sku-2", sku.Name)
		require.Equal(t, 2, sku.Spec.VCPU)
		require.Equal(t, 4, sku.Spec.Ram)
		require.Equal(t, testTenant, sku.Tenant)
	})

	t.Run("deletion_timestamp_is_propagated", func(t *testing.T) {
		cr := newInstanceSKUCR("compute-sku-3", "1", 8, 16, created)
		deletedAt := metav1.Time{Time: created.Add(time.Hour)}
		cr.SetDeletionTimestamp(&deletedAt)

		sku, err := InstanceSKUFromCR(cr)
		require.NoError(t, err)
		require.NotNil(t, sku.DeletedAt)
		require.Equal(t, deletedAt.UTC(), sku.DeletedAt.UTC())
	})

	t.Run("unsupported_type_errors", func(t *testing.T) {
		sku, err := InstanceSKUFromCR(&metav1.PartialObjectMetadata{})
		require.Error(t, err)
		require.Nil(t, sku)
	})
}

func TestInstanceSKUToCR(t *testing.T) {
	t.Run("populates_spec_and_identity", func(t *testing.T) {
		dom := &skudom.InstanceSKU{
			Spec: skudom.InstanceSKUSpec{VCPU: 4, Ram: 8},
		}
		dom.Name = "compute-sku-1"
		dom.ResourceVersion = "9"

		obj, err := InstanceSKUToCR(dom)
		require.NoError(t, err)

		cr, ok := obj.(*InstanceSKU)
		require.True(t, ok, "expected *InstanceSKU, got %T", obj)
		require.Equal(t, "compute-sku-1", cr.GetName())
		require.Equal(t, "9", cr.GetResourceVersion())
		require.Equal(t, InstanceSKUGVK, cr.GroupVersionKind())
		require.Equal(t, 4, cr.Spec.VCPU)
		require.Equal(t, 8, cr.Spec.Ram)
	})

	t.Run("spec_survives_round_trip", func(t *testing.T) {
		obj, err := InstanceSKUToCR(&skudom.InstanceSKU{
			Spec: skudom.InstanceSKUSpec{VCPU: 2, Ram: 4},
		})
		require.NoError(t, err)

		sku, err := InstanceSKUFromCR(obj)
		require.NoError(t, err)
		require.Equal(t, 2, sku.Spec.VCPU)
		require.Equal(t, 4, sku.Spec.Ram)
	})

	t.Run("nil_errors", func(t *testing.T) {
		obj, err := InstanceSKUToCR(nil)
		require.Error(t, err)
		require.Nil(t, obj)
	})
}
