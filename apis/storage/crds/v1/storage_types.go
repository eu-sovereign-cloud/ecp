package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	genv1 "github.com/eu-sovereign-cloud/ecp/apis/generated/types/storage/v1"
	"github.com/eu-sovereign-cloud/ecp/apis/storage"
)

// StorageSKUResource is the resource name for storage SKUs
const StorageSKUResource = "storage-skus"

var (
	StorageSKUGR  = schema.GroupResource{Group: storage.Group, Resource: StorageSKUResource}
	StorageSKUGVR = schema.GroupVersionResource{
		Group: storage.Group, Version: storage.Version, Resource: StorageSKUResource,
	}
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=storage-skus,scope=Cluster,shortName=storage-sku
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
