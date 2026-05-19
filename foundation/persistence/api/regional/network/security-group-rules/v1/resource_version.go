// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/network"
)

const (
	SecurityGroupRuleResource = "security-group-rules"
	SecurityGroupRuleKind     = "SecurityGroupRule"
)

var (
	SecurityGroupRuleGVR = schema.GroupVersionResource{
		Group: network.Group, Version: network.Version, Resource: SecurityGroupRuleResource,
	}
	SecurityGroupRuleGVK = schema.GroupVersionKind{
		Group: network.Group, Version: network.Version, Kind: SecurityGroupRuleKind,
	}
)
