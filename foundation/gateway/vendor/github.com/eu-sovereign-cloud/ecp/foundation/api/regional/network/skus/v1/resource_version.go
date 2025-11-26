// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/network"
)

// NetworkSKUResource is the resource name for network SKUs
const NetworkSKUResource = "network-skus"

var (
	NetworkSKUGR  = schema.GroupResource{Group: network.Group, Resource: NetworkSKUResource}
	NetworkSKUGVR = schema.GroupVersionResource{
		Group: network.Group, Version: network.Version, Resource: NetworkSKUResource,
	}
)
