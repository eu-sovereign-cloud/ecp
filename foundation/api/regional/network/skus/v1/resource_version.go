// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/network"
)

// SKUResource is the resource name for network SKUs.
const SKUResource = "skus"

var (
	SKUGR  = schema.GroupResource{Group: network.Group, Resource: SKUResource}
	SKUGVR = schema.GroupVersionResource{
		Group: network.Group, Version: network.Version, Resource: SKUResource,
	}
)
