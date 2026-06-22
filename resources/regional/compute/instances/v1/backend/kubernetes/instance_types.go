package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=instances,scope=Namespaced,shortName=instance
// +k8s:openapi-gen=true
// +ecp:conditioned

// Instance is the API for managing compute instances.
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       InstanceSpec    `json:"spec,omitempty"`
	CommonData genv1.CommonData      `json:"commonData,omitempty"`
	Status     *InstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instance{}, &InstanceList{})
}
