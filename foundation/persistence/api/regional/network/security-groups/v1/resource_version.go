// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/network"
)

const (
	SecurityGroupResource = "security-groups"
	SecurityGroupKind     = "SecurityGroup"
)

var (
	SecurityGroupGVR = schema.GroupVersionResource{
		Group: network.Group, Version: network.Version, Resource: SecurityGroupResource,
	}
	SecurityGroupGVK = schema.GroupVersionKind{
		Group: network.Group, Version: network.Version, Kind: SecurityGroupKind,
	}
)
