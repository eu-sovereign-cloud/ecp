package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=route-tables,scope=Namespaced,shortName=route-table
// +k8s:openapi-gen=true
// +ecp:conditioned

// RouteTable is the API for managing route tables.
type RouteTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.RouteTableSpec    `json:"spec,omitempty"`
	CommonData common.CommonData       `json:"commonData,omitempty"`
	Status     *genv1.RouteTableStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type RouteTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RouteTable `json:"items"`
}

func init() {
	network.SchemeBuilder.Register(&RouteTable{}, &RouteTableList{})
}
