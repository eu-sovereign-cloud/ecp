// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/apis/storage"
)

// XBlockStorageResource is the resource name for storage SKUs
const XBlockStorageResource = "xblock-storages"

var (
	XBlockStorageGR  = schema.GroupResource{Group: storage.Group, Resource: XBlockStorageResource}
	XBlockStorageGVR = schema.GroupVersionResource{
		Group: storage.Group, Version: storage.Version, Resource: XBlockStorageResource,
	}
)

type BlockStorageSpec struct {
	SizeGB   int    `json:"sizeGB"`
	SkuRef   string `json:"skuRef"`
	ImageRef string `json:"imageRef"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=xblock-storages,scope=Cluster,shortName=xblock
// +k8s:openapi-gen=true
// +crossbuilder:generate:xrd:claimNames:kind=BlockStorage,plural=block-storages

// XBlockStorage is the API for getting the block storages.
type XBlockStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BlockStorageSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type XBlockStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []XBlockStorage `json:"items"`
}

func init() {
	storage.SchemeBuilder.Register(&XBlockStorage{}, &XBlockStorageList{})
}
