package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BlockStorageSpec struct {
	SizeGB   int    `json:"sizeGB"`
	SkuRef   string `json:"skuRef"`
	ImageRef string `json:"imageRef"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=blockStorages,scope=Cluster,shortName=block
// +k8s:openapi-gen=true
// +crossbuilder:generate:xrd:claimNames:kind=BlockStorage,plural=blockStorages

// BlockStorage is the API for getting the block storages.
type BlockStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BlockStorageSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type BlockStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []BlockStorage `json:"items"`
}
