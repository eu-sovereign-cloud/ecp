package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=network-skus,scope=Namespaced,shortName=network-sku
// +k8s:openapi-gen=true

// NetworkSKU is the API for getting network SKUs information.
type NetworkSKU struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkSkuSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type NetworkSKUList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkSKU `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkSKU{}, &NetworkSKUList{})
}
