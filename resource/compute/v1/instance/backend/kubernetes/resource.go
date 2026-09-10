// +kubebuilder:object:generate=true
// +groupName=compute.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedomain "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

const (
	InstanceResource = instancedomain.Resource
	InstanceKind     = instancedomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: instancedomain.Group, Version: instancedomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	InstanceGVR = domain.GVR(instancedomain.Group, instancedomain.Version, instancedomain.Resource)
	InstanceGVK = domain.GVK(instancedomain.Group, instancedomain.Version, instancedomain.Kind)
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

	Spec       InstanceSpec        `json:"spec,omitempty"`
	CommonData schemav1.CommonData `json:"commonData,omitempty"`
	Status     *InstanceStatus     `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instance{}, &InstanceList{})
}
