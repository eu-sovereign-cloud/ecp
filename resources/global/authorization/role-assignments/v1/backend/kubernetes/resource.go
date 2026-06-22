// +kubebuilder:object:generate=true
// +groupName=authorization.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

const (
	Group   = "authorization.v1.secapi.cloud"
	Version = "v1"

	RoleAssignmentResource = "role-assignments"
	RoleAssignmentKind     = "RoleAssignment"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	RoleAssignmentGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: RoleAssignmentResource,
	}
	RoleAssignmentGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: RoleAssignmentKind,
	}
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
	CommonData genv1.CommonData      `json:"commonData,omitempty"`
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
