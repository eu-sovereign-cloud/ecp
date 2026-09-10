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
	subnetdomain "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

const (
	SubnetResource = subnetdomain.Resource
	SubnetKind     = subnetdomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: subnetdomain.Group, Version: subnetdomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	SubnetGVR = domain.GVR(subnetdomain.Group, subnetdomain.Version, subnetdomain.Resource)
	SubnetGVK = domain.GVK(subnetdomain.Group, subnetdomain.Version, subnetdomain.Kind)
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=subnets,scope=Namespaced,shortName=subnet
// +k8s:openapi-gen=true
// +ecp:conditioned

// Subnet is the API for managing network subnets.
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       SubnetSpec          `json:"spec,omitempty"`
	CommonData schemav1.CommonData `json:"commonData,omitempty"`
	Status     *SubnetStatus       `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type SubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Subnet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Subnet{}, &SubnetList{})
}
