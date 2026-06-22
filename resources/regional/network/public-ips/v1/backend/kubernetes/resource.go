// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
)

const (
	Group   = "network.v1.secapi.cloud"
	Version = "v1"

	PublicIpResource = "public-ips"
	PublicIpKind     = "PublicIp"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	PublicIpGVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: PublicIpResource}
	PublicIpGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: PublicIpKind}
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

	Spec       PublicIpSpec     `json:"spec,omitempty"`
	CommonData genv1.CommonData `json:"commonData,omitempty"`
	Status     *PublicIpStatus  `json:"status,omitempty"`
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
