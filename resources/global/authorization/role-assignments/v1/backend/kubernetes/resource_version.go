// +kubebuilder:object:generate=true
// +groupName=authorization.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "authorization.v1.secapi.cloud"
	Version = "v1"

	RoleAssignmentResource = "role-assignments"
	RoleAssignmentKind     = "RoleAssignment"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	RoleAssignmentGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: RoleAssignmentResource,
	}
	RoleAssignmentGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: RoleAssignmentKind,
	}
)
