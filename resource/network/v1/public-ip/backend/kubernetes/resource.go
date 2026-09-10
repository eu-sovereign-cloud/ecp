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
	publicipdomain "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

const (
	PublicIPResource = publicipdomain.Resource
	PublicIPKind     = publicipdomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: publicipdomain.Group, Version: publicipdomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	PublicIPGVR = domain.GVR(publicipdomain.Group, publicipdomain.Version, publicipdomain.Resource)
	PublicIPGVK = domain.GVK(publicipdomain.Group, publicipdomain.Version, publicipdomain.Kind)
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=public-ips,scope=Namespaced,shortName=public-ip
// +k8s:openapi-gen=true
// +ecp:conditioned

// PublicIP is the API for managing public IP addresses.
type PublicIP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       PublicIpSpec        `json:"spec,omitempty"`
	CommonData schemav1.CommonData `json:"commonData,omitempty"`
	Status     *PublicIpStatus     `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type PublicIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PublicIP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PublicIP{}, &PublicIPList{})
}
