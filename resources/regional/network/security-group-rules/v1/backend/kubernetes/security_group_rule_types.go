package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
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

	Spec       SecurityGroupRuleSpec    `json:"spec,omitempty"`
	CommonData genv1.CommonData               `json:"commonData,omitempty"`
	Status     *SecurityGroupRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type SecurityGroupRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecurityGroupRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecurityGroupRule{}, &SecurityGroupRuleList{})
}
