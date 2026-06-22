package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=role-assignments,scope=Namespaced,shortName=role-assignment
// +k8s:openapi-gen=true
// +ecp:conditioned

// RoleAssignment is the API for managing role assignments.
type RoleAssignment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.RoleAssignmentSpec    `json:"spec,omitempty"`
	CommonData genv1.CommonData            `json:"commonData,omitempty"`
	Status     *genv1.RoleAssignmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type RoleAssignmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RoleAssignment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RoleAssignment{}, &RoleAssignmentList{})
}
