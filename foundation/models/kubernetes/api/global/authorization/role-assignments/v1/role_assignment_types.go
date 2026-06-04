package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/global/authorization"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/generated/types"
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
	CommonData common.CommonData           `json:"commonData,omitempty"`
	Status     *genv1.RoleAssignmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type RoleAssignmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RoleAssignment `json:"items"`
}

func init() {
	authorization.SchemeBuilder.Register(&RoleAssignment{}, &RoleAssignmentList{})
}
