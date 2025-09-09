package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=secaregions,scope=Cluster,shortName=reg
// +k8s:openapi-gen=true

// SecaRegion is the API for getting the regions of a service.
type SecaRegion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RegionSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type SecaRegionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecaRegion `json:"items"`
}
