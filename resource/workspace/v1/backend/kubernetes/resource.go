// +kubebuilder:object:generate=true
// +groupName=workspace.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	workspacedomain "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

const (
	WorkspaceResource = workspacedomain.Resource
	WorkspaceKind     = workspacedomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: workspacedomain.Group, Version: workspacedomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	WorkspaceGVR = domain.GVR(workspacedomain.Group, workspacedomain.Version, workspacedomain.Resource)
	WorkspaceGVK = domain.GVK(workspacedomain.Group, workspacedomain.Version, workspacedomain.Kind)
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=workspaces,scope=Namespaced,shortName=workspace
// +kubebuilder:printcolumn:name="Region",type=string,JSONPath=`.region`
// +kubebuilder:selectablefield:JSONPath=`.region`
// +k8s:openapi-gen=true
// +ecp:conditioned

// Workspace is the API for getting the workspaces of a service.
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Region is the region this workspace belongs to, and part of its identity: the same name
	// in the same tenant is a different workspace in a different region, and each lives in its
	// own namespace. It is a field rather than only the internal region label so it is visible
	// in `kubectl get workspace` and selectable with a field selector; the label is still
	// written, because that is what the gateway's list filter selects on server-side.
	Region string `json:"region,omitempty"`

	// WorkspaceSpec is a type alias for map[string]string. Use the underlying
	// map type here to avoid a controller-gen v0.20 panic on type aliases.
	Spec       map[string]string   `json:"spec,omitempty"`
	CommonData schemav1.CommonData `json:"commonData,omitempty"`
	Status     *WorkspaceStatus    `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type WorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Workspace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workspace{}, &WorkspaceList{})
}
