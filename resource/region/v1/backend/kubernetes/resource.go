// +kubebuilder:object:generate=true
// +groupName=v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	regiondomain "github.com/eu-sovereign-cloud/ecp/resource/region/v1"
)

const (
	RegionResource = regiondomain.Resource
	RegionKind     = regiondomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: regiondomain.Group, Version: regiondomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	RegionGVR = domain.GVR(regiondomain.Group, regiondomain.Version, regiondomain.Resource)
	RegionGVK = domain.GVK(regiondomain.Group, regiondomain.Version, regiondomain.Kind)
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
