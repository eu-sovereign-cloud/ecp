package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=subnets,scope=Namespaced,shortName=subnet
// +k8s:openapi-gen=true
// +ecp:conditioned

// Subnet is the API for managing network subnets.
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       SubnetSpec       `json:"spec,omitempty"`
	CommonData genv1.CommonData `json:"commonData,omitempty"`
	Status     *SubnetStatus    `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type SubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Subnet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Subnet{}, &SubnetList{})
}
