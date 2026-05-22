package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/network"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=security-group-rules,scope=Namespaced,shortName=security-group-rule
// +k8s:openapi-gen=true
// +ecp:conditioned

// SecurityGroupRule is the API for managing security group rules.
type SecurityGroupRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.SecurityGroupRuleSpec    `json:"spec,omitempty"`
	CommonData common.CommonData              `json:"commonData,omitempty"`
	Status     *genv1.SecurityGroupRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type SecurityGroupRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecurityGroupRule `json:"items"`
}

func init() {
	network.SchemeBuilder.Register(&SecurityGroupRule{}, &SecurityGroupRuleList{})
}
