package v1

import (
	storage "github.com/eu-sovereign-cloud/ecp/apis/compute"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/apis/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=compute-skus,scope=Namespaced,shortName=compute-sku
// +k8s:openapi-gen=true

// Compute is the API for getting storage SKU information
type ComputeSKU struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// todo: use compute SKU spec. possibly add here
	// https://github.com/eu-sovereign-cloud/go-sdk/tree/main/pkg/spec/schema
	Spec genv1.StorageSkuSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type ComputeSKUList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ComputeSKU `json:"items"`
}

func init() {
	storage.SchemeBuilder.Register(&ComputeSKU{}, &ComputeSKUList{})
}
