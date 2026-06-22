package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=workspaces,scope=Namespaced,shortName=workspace
// +k8s:openapi-gen=true
// +ecp:conditioned

// Workspace is the API for getting the workspaces of a service.
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       WorkspaceSpec    `json:"spec,omitempty"`
	CommonData genv1.CommonData `json:"commonData,omitempty"`
	Status     *WorkspaceStatus `json:"status,omitempty"`
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
