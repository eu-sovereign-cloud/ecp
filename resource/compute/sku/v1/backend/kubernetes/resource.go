// +kubebuilder:object:generate=true
// +groupName=compute.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "compute.v1.secapi.cloud"
	Version = "v1"

	InstanceSKUResource = "skus"
	InstanceSKUKind     = "InstanceSKU"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	InstanceSKUGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: InstanceSKUResource,
	}
	InstanceSKUGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: InstanceSKUKind,
	}
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=skus,scope=Namespaced,shortName=instance-sku
// +k8s:openapi-gen=true

// InstanceSKU is the API for getting compute instance SKU information.
type InstanceSKU struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec InstanceSkuSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type InstanceSKUList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []InstanceSKU `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstanceSKU{}, &InstanceSKUList{})
}
