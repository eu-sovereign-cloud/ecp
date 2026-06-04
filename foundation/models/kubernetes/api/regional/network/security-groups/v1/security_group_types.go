package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=security-groups,scope=Namespaced,shortName=security-group
// +k8s:openapi-gen=true
// +ecp:conditioned

// SecurityGroup is the API for managing security groups.
type SecurityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.SecurityGroupSpec    `json:"spec,omitempty"`
	CommonData common.CommonData          `json:"commonData,omitempty"`
	Status     *genv1.SecurityGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type SecurityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecurityGroup `json:"items"`
}

func init() {
	network.SchemeBuilder.Register(&SecurityGroup{}, &SecurityGroupList{})
}
