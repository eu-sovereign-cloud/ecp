package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=public-ips,scope=Namespaced,shortName=public-ip
// +k8s:openapi-gen=true
// +ecp:conditioned

// PublicIp is the API for managing public IP addresses.
type PublicIp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.PublicIpSpec    `json:"spec,omitempty"`
	CommonData genv1.CommonData      `json:"commonData,omitempty"`
	Status     *genv1.PublicIpStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type PublicIpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PublicIp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PublicIp{}, &PublicIpList{})
}
