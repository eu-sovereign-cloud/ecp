package v1

import (
	// xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// // StorageStatus defines the observed state of Storage.
// type StorageStatus struct {
// 	xpv1.ResourceStatus `json:",inline"`
// 	// Add your observed state fields here. For example:
// 	// AtProvider contains the observed state of the external resource.
// 	AtProvider struct {
// 		State string `json:"state,omitempty"`
// 	} `json:"atProvider,omitempty"`
// }

// Storage is the Schema for the storages API. It represents a composite resource.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,secapi.eu}
// +crossbuilder:generate:xrd:claimNames:kind=Storage,plural=Storages

type Storage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec StorageSpec `json:"spec,omitempty"`
	// Status StorageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StorageList contains a list of Storage
type StorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Storage `json:"items"`
}

// Storage type metadata.
var (
	StorageKind             = "Storage"
	StorageGroupKind        = schema.GroupKind{Group: Group, Kind: StorageKind}.String()
	StorageKindAPIVersion   = StorageKind + "." + GroupVersion.String()
	StorageGroupVersionKind = GroupVersion.WithKind(StorageKind)
)

func init() {
	SchemeBuilder.Register(&Storage{}, &StorageList{})
}
