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
	internetgatewaydomain "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

const (
	InternetGatewayResource = internetgatewaydomain.Resource
	InternetGatewayKind     = internetgatewaydomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: internetgatewaydomain.Group, Version: internetgatewaydomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	InternetGatewayGVR = domain.GVR(internetgatewaydomain.Group, internetgatewaydomain.Version, internetgatewaydomain.Resource)
	InternetGatewayGVK = domain.GVK(internetgatewaydomain.Group, internetgatewaydomain.Version, internetgatewaydomain.Kind)
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=internet-gateways,scope=Namespaced,shortName=internet-gateway
// +k8s:openapi-gen=true
// +ecp:conditioned

// InternetGateway is the API for managing internet gateways.
type InternetGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       InternetGatewaySpec    `json:"spec,omitempty"`
	CommonData schemav1.CommonData    `json:"commonData,omitempty"`
	Status     *InternetGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type InternetGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []InternetGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InternetGateway{}, &InternetGatewayList{})
}
