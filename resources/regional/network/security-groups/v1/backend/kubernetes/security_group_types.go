package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=security-groups,scope=Namespaced,shortName=security-group
// +k8s:openapi-gen=true
// +ecp:conditioned

// SecurityGroup is the API for managing security groups.
type SecurityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       SecurityGroupSpec    `json:"spec,omitempty"`
	CommonData genv1.CommonData           `json:"commonData,omitempty"`
	Status     *SecurityGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type SecurityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecurityGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecurityGroup{}, &SecurityGroupList{})
}
