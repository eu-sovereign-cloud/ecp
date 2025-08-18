package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=regions,scope=Cluster,singular=regions
// +k8s:openapi-gen=true

// Regions is the API for getting the regions of a service.
type Regions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Status reflects the observed state of a SGDbOps.
	Spec RegionSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type RegionsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RegionSpec `json:"items"`
}
