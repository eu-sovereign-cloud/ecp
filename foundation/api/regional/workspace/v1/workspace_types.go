package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/common"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=workspaces,scope=Namespaced,shortName=workspace
// +k8s:openapi-gen=true

// Workspace is the API for getting the workspaces of a service.
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec               v1.WorkspaceSpec          `json:"spec,omitempty"`
	RegionalCommonData common.RegionalCommonData `json:"regionalCommonData,omitempty"`
}

// +kubebuilder:object:root=true

type WorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Workspace `json:"items"`
}
