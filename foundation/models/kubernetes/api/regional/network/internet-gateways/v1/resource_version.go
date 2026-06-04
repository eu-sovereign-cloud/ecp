// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network"
)

const (
	InternetGatewayResource = "internet-gateways"
	InternetGatewayKind     = "InternetGateway"
)

var (
	InternetGatewayGVR = schema.GroupVersionResource{
		Group: network.Group, Version: network.Version, Resource: InternetGatewayResource,
	}
	InternetGatewayGVK = schema.GroupVersionKind{
		Group: network.Group, Version: network.Version, Kind: InternetGatewayKind,
	}
)
