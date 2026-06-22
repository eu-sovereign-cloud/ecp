// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "network.v1.secapi.cloud"
	Version = "v1"

	SecurityGroupRuleResource = "security-group-rules"
	SecurityGroupRuleKind     = "SecurityGroupRule"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	SecurityGroupRuleGVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: SecurityGroupRuleResource}
	SecurityGroupRuleGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: SecurityGroupRuleKind}
)
