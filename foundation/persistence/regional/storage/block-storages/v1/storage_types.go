package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/generated/types"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/regional/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/regional/storage"
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

	Spec               genv1.BlockStorageSpec    `json:"spec,omitempty"`
	RegionalCommonData common.RegionalCommonData `json:"regionalCommonData,omitempty"`
	Status             *genv1.BlockStorageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type BlockStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []BlockStorage `json:"items"`
}

func init() {
	storage.SchemeBuilder.Register(&BlockStorage{}, &BlockStorageList{})
}
