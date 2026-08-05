package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"

	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku/backend/kubernetes"
)

func TestNetworkSKUToCR(t *testing.T) {
	t.Run("populates_spec_and_identity", func(t *testing.T) {
		dom := &skudom.NetworkSKU{
			Spec: skudom.NetworkSKUSpec{Bandwidth: 5000, Packets: 40000},
		}
		dom.Name = "seca.n5k"
		dom.ResourceVersion = "7"

		obj, err := NetworkSKUToCR(dom)
		require.NoError(t, err)

		cr, ok := obj.(*NetworkSKU)
		require.True(t, ok, "expected *NetworkSKU, got %T", obj)
		require.Equal(t, "seca.n5k", cr.GetName())
		require.Equal(t, "7", cr.GetResourceVersion())
		require.Equal(t, NetworkSKUGVK, cr.GroupVersionKind())
		require.Equal(t, 5000, cr.Spec.Bandwidth)
		require.Equal(t, 40000, cr.Spec.Packets)
	})

	t.Run("spec_survives_round_trip", func(t *testing.T) {
		obj, err := NetworkSKUToCR(&skudom.NetworkSKU{
			Spec: skudom.NetworkSKUSpec{Bandwidth: 10000, Packets: 80000},
		})
		require.NoError(t, err)

		sku, err := NetworkSKUFromCR(obj)
		require.NoError(t, err)
		require.Equal(t, 10000, sku.Spec.Bandwidth)
		require.Equal(t, 80000, sku.Spec.Packets)
	})

	t.Run("identity_is_read_from_internal_labels", func(t *testing.T) {
		cr := &NetworkSKU{
			ObjectMeta: metav1.ObjectMeta{
				Name: "seca.n1k",
				Labels: map[string]string{
					// The provider is stored with "/" encoded as "_", since "/"
					// is not a legal label value character.
					k8slabels.InternalProviderLabel: "seca.network_v1",
					k8slabels.InternalRegionLabel:   "eu-central-1",
					k8slabels.InternalTenantLabel:   "tn-1",
				},
			},
			Spec: NetworkSkuSpec{Bandwidth: 1000, Packets: 10000},
		}

		sku, err := NetworkSKUFromCR(cr)
		require.NoError(t, err)
		require.Equal(t, "seca.network/v1", sku.Provider)
		require.Equal(t, "eu-central-1", sku.Region)
		require.Equal(t, "tn-1", sku.Tenant)
	})

	t.Run("nil_errors", func(t *testing.T) {
		obj, err := NetworkSKUToCR(nil)
		require.Error(t, err)
		require.Nil(t, obj)
	})

	t.Run("unsupported_type_errors", func(t *testing.T) {
		sku, err := NetworkSKUFromCR(&metav1.PartialObjectMetadata{})
		require.Error(t, err)
		require.Nil(t, sku)
	})
}
