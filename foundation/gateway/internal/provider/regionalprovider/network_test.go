package regionalprovider

import (
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/network/skus/v1"

	generatedv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
)

// --- Helpers ---

// newNetworkSKUCR constructs a typed NetworkSKU CR.
func newNetworkSKUCR(name, tenant string, labels map[string]string, bandwidth, packets int, setVersionAndTimestamp bool) *skuv1.NetworkSKU {
	if labels == nil {
		labels = map[string]string{}
	}
	cr := &skuv1.NetworkSKU{
		TypeMeta:   metav1.TypeMeta{Kind: "NetworkSKU", APIVersion: fmt.Sprintf("%s/%s", skuv1.NetworkSKUGVR.Group, skuv1.NetworkSKUGVR.Version)},
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels, Namespace: tenant},
		Spec:       generatedv1.NetworkSkuSpec{Bandwidth: bandwidth, Packets: packets},
	}
	if setVersionAndTimestamp {
		cr.SetCreationTimestamp(metav1.Time{Time: time.Unix(1700000000, 0)})
		cr.SetResourceVersion("1")
	}
	return cr
}

// --- Tests ---

func TestNetworkController_ListSKUs(t *testing.T) {
	// TODO implement me
}

func TestNetworkController_GetSKU(t *testing.T) {
	// TODO implement me
}
