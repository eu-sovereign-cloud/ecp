package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
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

	Spec       RouteTableSpec    `json:"spec,omitempty"`
	CommonData genv1.CommonData        `json:"commonData,omitempty"`
	Status     *RouteTableStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type RouteTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RouteTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RouteTable{}, &RouteTableList{})
}
