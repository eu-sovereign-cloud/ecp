// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

const (
	Group   = "network.v1.secapi.cloud"
	Version = "v1"

	InternetGatewayResource = "internet-gateways"
	InternetGatewayKind     = "InternetGateway"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	InternetGatewayGVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: InternetGatewayResource}
	InternetGatewayGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: InternetGatewayKind}
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
	CommonData genv1.CommonData       `json:"commonData,omitempty"`
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
