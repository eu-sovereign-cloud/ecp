// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "network.v1.secapi.cloud"
	Version = "v1"

	NetworkSKUResource = "network-skus"
	NetworkSKUKind     = "NetworkSKU"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	NetworkSKUGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: NetworkSKUResource,
	}
	NetworkSKUGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: NetworkSKUKind,
	}
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=network-skus,scope=Namespaced,shortName=network-sku
// +k8s:openapi-gen=true

// NetworkSKU is the API for getting network SKUs information.
type NetworkSKU struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkSkuSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type NetworkSKUList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkSKU `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkSKU{}, &NetworkSKUList{})
}
