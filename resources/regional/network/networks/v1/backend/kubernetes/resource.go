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

	NetworkResource = "networks"
	NetworkKind     = "Network"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	NetworkGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: NetworkResource,
	}
	NetworkGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: NetworkKind,
	}
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=networks,scope=Namespaced,shortName=network
// +k8s:openapi-gen=true
// +ecp:conditioned

// Network is the API for managing virtual networks.
type Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       NetworkSpec      `json:"spec,omitempty"`
	CommonData genv1.CommonData `json:"commonData,omitempty"`
	Status     *NetworkStatus   `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Network `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Network{}, &NetworkList{})
}
