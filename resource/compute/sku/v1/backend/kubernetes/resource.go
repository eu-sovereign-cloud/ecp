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

	InstanceSkuResource = "skus"
	InstanceSkuKind     = "InstanceSku"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	InstanceSkuGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: InstanceSkuResource,
	}
	InstanceSkuGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: InstanceSkuKind,
	}
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=skus,scope=Namespaced,shortName=instance-sku
// +k8s:openapi-gen=true

// InstanceSku is the API for getting compute instance SKU information.
type InstanceSku struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec InstanceSkuSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type InstanceSkuList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []InstanceSku `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstanceSku{}, &InstanceSkuList{})
}
