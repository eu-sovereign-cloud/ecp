// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	nicdomain "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

const (
	NICResource = nicdomain.Resource
	NICKind     = nicdomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: nicdomain.Group, Version: nicdomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	NICGVR = domain.GVR(nicdomain.Group, nicdomain.Version, nicdomain.Resource)
	NICGVK = domain.GVK(nicdomain.Group, nicdomain.Version, nicdomain.Kind)
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nics,scope=Namespaced,shortName=nic
// +k8s:openapi-gen=true
// +ecp:conditioned

// NIC is the API for managing network interface cards.
type NIC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       NicSpec             `json:"spec,omitempty"`
	CommonData schemav1.CommonData `json:"commonData,omitempty"`
	Status     *NicStatus          `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type NICList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NIC `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NIC{}, &NICList{})
}
