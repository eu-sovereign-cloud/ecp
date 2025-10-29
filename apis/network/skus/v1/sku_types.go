package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/apis/generated/types"
	network "github.com/eu-sovereign-cloud/ecp/apis/network"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=network-skus,scope=Cluster,shortName=network-sku
// +k8s:openapi-gen=true

// NetworkSKU is the API for getting network SKU information
type NetworkSKU struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec genv1.NetworkSkuSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type NetworkSKUList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkSKU `json:"items"`
}

func init() {
	network.SchemeBuilder.Register(&NetworkSKU{}, &NetworkSKUList{})
}
