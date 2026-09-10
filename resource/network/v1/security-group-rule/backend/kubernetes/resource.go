// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupruledomain "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

const (
	SecurityGroupRuleResource = securitygroupruledomain.Resource
	SecurityGroupRuleKind     = securitygroupruledomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: securitygroupruledomain.Group, Version: securitygroupruledomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	SecurityGroupRuleGVR = domain.GVR(securitygroupruledomain.Group, securitygroupruledomain.Version, securitygroupruledomain.Resource)
	SecurityGroupRuleGVK = domain.GVK(securitygroupruledomain.Group, securitygroupruledomain.Version, securitygroupruledomain.Kind)
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
