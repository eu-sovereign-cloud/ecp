// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network"
)

const (
	PublicIpResource = "public-ips"
	PublicIpKind     = "PublicIp"
)

var (
	PublicIpGVR = schema.GroupVersionResource{
		Group: network.Group, Version: network.Version, Resource: PublicIpResource,
	}
	PublicIpGVK = schema.GroupVersionKind{
		Group: network.Group, Version: network.Version, Kind: PublicIpKind,
	}
)
