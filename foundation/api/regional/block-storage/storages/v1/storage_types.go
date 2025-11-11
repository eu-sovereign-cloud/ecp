package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	storage "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/block-storage"
	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/common"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=storages,scope=Namespaced,shortName=storage
// +k8s:openapi-gen=true

// Storage is the API for getting storage SKU information
type Storage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec               genv1.BlockStorageSpec    `json:"spec,omitempty"`
	RegionalComonSpec  common.RegionalCommonSpec `json:"regionalComonSpec,omitempty"`
	RegionalCommonData common.RegionalCommonData `json:"regionalCommonData,omitempty"`
}

// +kubebuilder:object:root=true

type StorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Storage `json:"items"`
}

func init() {
	storage.SchemeBuilder.Register(&Storage{}, &StorageList{})
}
