// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	blockstoragedomain "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

const (
	BlockStorageResource = blockstoragedomain.Resource
	BlockStorageKind     = blockstoragedomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: blockstoragedomain.Group, Version: blockstoragedomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	BlockStorageGVR = domain.GVR(blockstoragedomain.Group, blockstoragedomain.Version, blockstoragedomain.Resource)
	BlockStorageGVK = domain.GVK(blockstoragedomain.Group, blockstoragedomain.Version, blockstoragedomain.Kind)
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
