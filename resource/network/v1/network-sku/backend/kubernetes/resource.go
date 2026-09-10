// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	networkskudomain "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
)

const (
	NetworkSKUResource = networkskudomain.Resource
	NetworkSKUKind     = networkskudomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: networkskudomain.Group, Version: networkskudomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	NetworkSKUGVR = domain.GVR(networkskudomain.Group, networkskudomain.Version, networkskudomain.Resource)
	NetworkSKUGVK = domain.GVK(networkskudomain.Group, networkskudomain.Version, networkskudomain.Kind)
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
