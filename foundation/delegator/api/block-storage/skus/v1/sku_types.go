package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	storage "github.com/eu-sovereign-cloud/ecp/foundation/delegator/api/block-storage"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/delegator/api/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=storage-skus,scope=Namespaced,shortName=storage-sku
// +k8s:openapi-gen=true

// StorageSKU is the API for getting storage SKU information
type StorageSKU struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec genv1.StorageSkuSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type StorageSKUList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []StorageSKU `json:"items"`
}

func init() {
	storage.SchemeBuilder.Register(&StorageSKU{}, &StorageSKUList{})
}
