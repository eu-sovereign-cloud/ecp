package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/generated/types"
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

	Spec       genv1.InternetGatewaySpec    `json:"spec,omitempty"`
	CommonData common.CommonData            `json:"commonData,omitempty"`
	Status     *genv1.InternetGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type InternetGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []InternetGateway `json:"items"`
}

func init() {
	network.SchemeBuilder.Register(&InternetGateway{}, &InternetGatewayList{})
}
