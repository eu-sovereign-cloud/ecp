package network

import (
	"testing"
	"time"

	generatedv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/network/skus/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newNetworkSKUCR constructs a typed NetworkSKU CR.
func newNetworkSKUCR(name, tenant string, labels map[string]string, bandwidth, packets int, setVersionAndTimestamp bool) *skuv1.NetworkSKU {
	if labels == nil {
		labels = map[string]string{}
	}
	cr := &skuv1.NetworkSKU{
		TypeMeta:   metav1.TypeMeta{Kind: "NetworkSKU", APIVersion: skuv1.NetworkSKUGVR.GroupVersion().String()},
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels, Namespace: tenant},
		Spec:       generatedv1.NetworkSkuSpec{Bandwidth: bandwidth, Packets: packets},
	}
	if setVersionAndTimestamp {
		cr.SetCreationTimestamp(metav1.Time{Time: time.Unix(1700000000, 0)})
		cr.SetResourceVersion("1")
	}
	return cr
}

func TestNetworkController_GetSKU(t *testing.T) {
	// TODO implement me
}
