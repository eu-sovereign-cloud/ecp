package v1

import (
	regionv1 "github.com/eu-sovereign-cloud/ecp/apis/generated/types/region/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=regions,scope=Cluster,singular=regions
// +k8s:openapi-gen=true

// Regions is the API for getting the regions of a service.
type Regions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec regionv1.RegionSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type RegionsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []regionv1.RegionSpec `json:"items"`
}
