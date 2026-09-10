// +kubebuilder:object:generate=true
// +groupName=authorization.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	roledomain "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

const (
	RoleResource = roledomain.Resource
	RoleKind     = roledomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: roledomain.Group, Version: roledomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	RoleGVR = domain.GVR(roledomain.Group, roledomain.Version, roledomain.Resource)
	RoleGVK = domain.GVK(roledomain.Group, roledomain.Version, roledomain.Kind)
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

	Spec       RoleSpec            `json:"spec,omitempty"`
	CommonData schemav1.CommonData `json:"commonData,omitempty"`
	Status     *RoleStatus         `json:"status,omitempty"`
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
