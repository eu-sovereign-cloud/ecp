// +kubebuilder:object:generate=true
// +groupName=compute.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instanceskudomain "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku"
)

const (
	InstanceSKUResource = instanceskudomain.Resource
	InstanceSKUKind     = instanceskudomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: instanceskudomain.Group, Version: instanceskudomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	InstanceSKUGVR = domain.GVR(instanceskudomain.Group, instanceskudomain.Version, instanceskudomain.Resource)
	InstanceSKUGVK = domain.GVK(instanceskudomain.Group, instanceskudomain.Version, instanceskudomain.Kind)
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
