package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=public-ips,scope=Namespaced,shortName=public-ip
// +k8s:openapi-gen=true
// +ecp:conditioned

// PublicIp is the API for managing public IP addresses.
type PublicIp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.PublicIpSpec    `json:"spec,omitempty"`
	CommonData common.CommonData     `json:"commonData,omitempty"`
	Status     *genv1.PublicIpStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type PublicIpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PublicIp `json:"items"`
}

func init() {
	network.SchemeBuilder.Register(&PublicIp{}, &PublicIpList{})
}
