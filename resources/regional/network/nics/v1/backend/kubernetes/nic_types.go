package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nics,scope=Namespaced,shortName=nic
// +k8s:openapi-gen=true
// +ecp:conditioned

// Nic is the API for managing network interface cards.
type Nic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.NicSpec    `json:"spec,omitempty"`
	CommonData genv1.CommonData `json:"commonData,omitempty"`
	Status     *genv1.NicStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type NicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Nic `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nic{}, &NicList{})
}
