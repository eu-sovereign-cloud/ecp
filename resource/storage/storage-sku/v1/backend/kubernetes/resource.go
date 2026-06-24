// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "storage.v1.secapi.cloud"
	Version = "v1"

	StorageSKUResource = "skus"
	StorageSKUKind     = "StorageSKU"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	StorageSKUGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: StorageSKUResource,
	}
	StorageSKUGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: StorageSKUKind,
	}
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
