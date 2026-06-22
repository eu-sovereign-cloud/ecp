// +kubebuilder:object:generate=true
// +groupName=network.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "network.v1.secapi.cloud"
	Version = "v1"

	RouteTableResource = "route-tables"
	RouteTableKind     = "RouteTable"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	RouteTableGVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: RouteTableResource}
	RouteTableGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: RouteTableKind}
)
