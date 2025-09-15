package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/eu-sovereign-cloud/ecp/apis/generated/types/region/v1"
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

	Items []v1.RegionSpec `json:"items"`
}
