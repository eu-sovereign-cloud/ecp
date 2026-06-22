// +kubebuilder:object:generate=true
// +groupName=authorization.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

const (
	Group   = "authorization.v1.secapi.cloud"
	Version = "v1"

	RoleResource = "roles"
	RoleKind     = "Role"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	RoleGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: RoleResource,
	}
	RoleGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: RoleKind,
	}
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
