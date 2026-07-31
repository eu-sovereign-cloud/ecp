package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
