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
	routetabledomain "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

const (
	RouteTableResource = routetabledomain.Resource
	RouteTableKind     = routetabledomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: routetabledomain.Group, Version: routetabledomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	RouteTableGVR = domain.GVR(routetabledomain.Group, routetabledomain.Version, routetabledomain.Resource)
	RouteTableGVK = domain.GVK(routetabledomain.Group, routetabledomain.Version, routetabledomain.Kind)
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=route-tables,scope=Namespaced,shortName=route-table
// +k8s:openapi-gen=true
// +ecp:conditioned

// RouteTable is the API for managing route tables.
type RouteTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       RouteTableSpec      `json:"spec,omitempty"`
	CommonData schemav1.CommonData `json:"commonData,omitempty"`
	Status     *RouteTableStatus   `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type RouteTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RouteTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RouteTable{}, &RouteTableList{})
}
