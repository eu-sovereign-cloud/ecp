// +kubebuilder:object:generate=true
// +groupName=v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "v1.secapi.cloud"
	Version = "v1"

	RegionResource = "regions"
	RegionKind     = "Region"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	RegionGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: RegionResource,
	}
	RegionGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: RegionKind,
	}
)
