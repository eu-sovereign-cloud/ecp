// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	storageskudomain "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku"
)

const (
	StorageSKUResource = storageskudomain.Resource
	StorageSKUKind     = storageskudomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: storageskudomain.Group, Version: storageskudomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	StorageSKUGVR = domain.GVR(storageskudomain.Group, storageskudomain.Version, storageskudomain.Resource)
	StorageSKUGVK = domain.GVK(storageskudomain.Group, storageskudomain.Version, storageskudomain.Kind)
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
