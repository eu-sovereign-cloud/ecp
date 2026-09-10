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
	securitygroupdomain "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

const (
	SecurityGroupResource = securitygroupdomain.Resource
	SecurityGroupKind     = securitygroupdomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: securitygroupdomain.Group, Version: securitygroupdomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	SecurityGroupGVR = domain.GVR(securitygroupdomain.Group, securitygroupdomain.Version, securitygroupdomain.Resource)
	SecurityGroupGVK = domain.GVK(securitygroupdomain.Group, securitygroupdomain.Version, securitygroupdomain.Kind)
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

	Spec       SecurityGroupSpec    `json:"spec,omitempty"`
	CommonData schemav1.CommonData  `json:"commonData,omitempty"`
	Status     *SecurityGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type SecurityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecurityGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecurityGroup{}, &SecurityGroupList{})
}
