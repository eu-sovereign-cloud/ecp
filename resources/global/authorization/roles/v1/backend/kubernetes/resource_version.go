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

	RoleResource = "roles"
	RoleKind     = "Role"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	RoleGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: RoleResource,
	}
	RoleGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: RoleKind,
	}
)
