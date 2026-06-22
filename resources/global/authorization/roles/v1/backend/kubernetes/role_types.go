package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=roles,scope=Namespaced,shortName=role
// +k8s:openapi-gen=true
// +ecp:conditioned

// Role is the API for managing authorization roles.
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       RoleSpec    `json:"spec,omitempty"`
	CommonData genv1.CommonData  `json:"commonData,omitempty"`
	Status     *RoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type RoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Role `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Role{}, &RoleList{})
}
