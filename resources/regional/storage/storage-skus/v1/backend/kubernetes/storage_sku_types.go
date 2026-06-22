package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=skus,scope=Namespaced,shortName=sku
// +k8s:openapi-gen=true

// StorageSKU is the API for getting storage SKUs information.
type StorageSKU struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec StorageSkuSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type StorageSKUList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []StorageSKU `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageSKU{}, &StorageSKUList{})
}
