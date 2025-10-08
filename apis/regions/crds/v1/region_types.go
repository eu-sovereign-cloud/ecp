// +kubebuilder:object:generate=true
// +groupName=v1.secapi.cloud
// +versionName=v1

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "github.com/eu-sovereign-cloud/ecp/apis/generated/types/region/v1"
	"github.com/eu-sovereign-cloud/ecp/apis/regions"
)

// RegionResource is the resource name for regions
const RegionResource = "regions"

var (
	RegionGR  = schema.GroupResource{Group: regions.Group, Resource: RegionResource}
	RegionGVR = schema.GroupVersionResource{Group: regions.Group, Version: regions.Version, Resource: RegionResource}
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=regions,scope=Cluster,shortName=reg
// +k8s:openapi-gen=true

// Region is the API for getting the regions of a service.
type Region struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec v1.RegionSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type RegionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Region `json:"items"`
}

func init() {
	regions.SchemeBuilder.Register(&Region{}, &RegionList{})
}
