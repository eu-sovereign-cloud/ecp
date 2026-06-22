package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=images,scope=Namespaced,shortName=image
// +k8s:openapi-gen=true
// +ecp:conditioned

// Image is the API for managing compute images.
type Image struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       ImageSpec        `json:"spec,omitempty"`
	CommonData genv1.CommonData `json:"commonData,omitempty"`
	Status     *ImageStatus     `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Image `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Image{}, &ImageList{})
}
