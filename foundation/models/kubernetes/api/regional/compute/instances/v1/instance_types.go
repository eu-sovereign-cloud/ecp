package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/compute"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=instances,scope=Namespaced,shortName=instance
// +k8s:openapi-gen=true
// +ecp:conditioned

// Instance is the API for managing compute instances.
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.InstanceSpec    `json:"spec,omitempty"`
	CommonData common.CommonData     `json:"commonData,omitempty"`
	Status     *genv1.InstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Instance `json:"items"`
}

func init() {
	compute.SchemeBuilder.Register(&Instance{}, &InstanceList{})
}
