package v1

import (
	storage "github.com/eu-sovereign-cloud/ecp/apis/compute"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/apis/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=instance-skus,scope=Namespaced,shortName=instance-sku
// +k8s:openapi-gen=true

// Compute is the API for getting storage SKU information
type InstanceSKU struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec genv1.InstanceSkuSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type InstanceSKUList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []InstanceSKU `json:"items"`
}

func init() {
	storage.SchemeBuilder.Register(&InstanceSKU{}, &InstanceSKUList{})
}
