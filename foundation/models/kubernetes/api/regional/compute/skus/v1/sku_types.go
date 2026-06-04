package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/compute"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=skus,scope=Namespaced,shortName=instance-sku
// +k8s:openapi-gen=true

// InstanceSku is the API for getting compute instance SKU information.
type InstanceSku struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec genv1.InstanceSkuSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type InstanceSkuList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []InstanceSku `json:"items"`
}

func init() {
	compute.SchemeBuilder.Register(&InstanceSku{}, &InstanceSkuList{})
}
