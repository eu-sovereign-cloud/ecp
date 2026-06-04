// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network"
)

const (
	SubnetResource = "subnets"
	SubnetKind     = "Subnet"
)

var (
	SubnetGVR = schema.GroupVersionResource{
		Group: network.Group, Version: network.Version, Resource: SubnetResource,
	}
	SubnetGVK = schema.GroupVersionKind{
		Group: network.Group, Version: network.Version, Kind: SubnetKind,
	}
)
