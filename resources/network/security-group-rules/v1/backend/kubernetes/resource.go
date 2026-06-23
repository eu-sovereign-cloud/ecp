// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
)

const (
	Group   = "network.v1.secapi.cloud"
	Version = "v1"

	SecurityGroupRuleResource = "security-group-rules"
	SecurityGroupRuleKind     = "SecurityGroupRule"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	SecurityGroupRuleGVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: SecurityGroupRuleResource}
	SecurityGroupRuleGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: SecurityGroupRuleKind}
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
	CommonData schemav1.CommonData      `json:"commonData,omitempty"`
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
