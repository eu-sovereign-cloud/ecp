// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
)

const (
	Group   = "storage.v1.secapi.cloud"
	Version = "v1"

	BlockStorageResource = "block-storages"
	BlockStorageKind     = "BlockStorage"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	BlockStorageGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: BlockStorageResource,
	}
	BlockStorageGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: BlockStorageKind,
	}
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=block-storages,scope=Namespaced,shortName=block-storage
// +k8s:openapi-gen=true
// +ecp:conditioned

// BlockStorage is the API for getting storage block-storage instances information.
type BlockStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       BlockStorageSpec    `json:"spec,omitempty"`
	CommonData schemav1.CommonData `json:"commonData,omitempty"`
	Status     *BlockStorageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type BlockStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []BlockStorage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BlockStorage{}, &BlockStorageList{})
}
