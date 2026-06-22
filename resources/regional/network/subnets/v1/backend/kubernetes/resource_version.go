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

	SubnetResource = "subnets"
	SubnetKind     = "Subnet"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	SubnetGVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: SubnetResource}
	SubnetGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: SubnetKind}
)
