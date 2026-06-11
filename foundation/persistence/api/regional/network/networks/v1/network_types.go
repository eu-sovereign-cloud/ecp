package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/network"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=networks,scope=Namespaced,shortName=network
// +k8s:openapi-gen=true
// +ecp:conditioned

// Network is the API for managing virtual networks.
type Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.NetworkSpec    `json:"spec,omitempty"`
	CommonData common.CommonData    `json:"commonData,omitempty"`
	Status     *genv1.NetworkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Network `json:"items"`
}

func init() {
	network.SchemeBuilder.Register(&Network{}, &NetworkList{})
}
