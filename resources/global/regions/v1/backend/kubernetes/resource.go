// +kubebuilder:object:generate=true
// +groupName=v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=regions,scope=Cluster,shortName=reg
// +k8s:openapi-gen=true

// Region is the API for getting the regions of a service.
type Region struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RegionSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type RegionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Region `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Region{}, &RegionList{})
}
