// +kubebuilder:object:generate=true
// +groupName=compute.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/compute"
)

const (
	InstanceSkuResource = "skus"
	InstanceSkuKind     = "InstanceSku"
)

var (
	InstanceSkuGVR = schema.GroupVersionResource{
		Group: compute.Group, Version: compute.Version, Resource: InstanceSkuResource,
	}
	InstanceSkuGVK = schema.GroupVersionKind{
		Group: compute.Group, Version: compute.Version, Kind: InstanceSkuKind,
	}
)
