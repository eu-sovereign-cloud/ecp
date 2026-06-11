package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/network"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=subnets,scope=Namespaced,shortName=subnet
// +k8s:openapi-gen=true
// +ecp:conditioned

// Subnet is the API for managing network subnets.
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.SubnetSpec    `json:"spec,omitempty"`
	CommonData common.CommonData   `json:"commonData,omitempty"`
	Status     *genv1.SubnetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type SubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Subnet `json:"items"`
}

func init() {
	network.SchemeBuilder.Register(&Subnet{}, &SubnetList{})
}
