// +kubebuilder:object:generate=true
// +groupName=authorization.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	roleassignmentdomain "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

const (
	RoleAssignmentResource = roleassignmentdomain.Resource
	RoleAssignmentKind     = roleassignmentdomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: roleassignmentdomain.Group, Version: roleassignmentdomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	RoleAssignmentGVR = domain.GVR(roleassignmentdomain.Group, roleassignmentdomain.Version, roleassignmentdomain.Resource)
	RoleAssignmentGVK = domain.GVK(roleassignmentdomain.Group, roleassignmentdomain.Version, roleassignmentdomain.Kind)
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

	Spec       RoleAssignmentSpec    `json:"spec,omitempty"`
	CommonData schemav1.CommonData   `json:"commonData,omitempty"`
	Status     *RoleAssignmentStatus `json:"status,omitempty"`
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
