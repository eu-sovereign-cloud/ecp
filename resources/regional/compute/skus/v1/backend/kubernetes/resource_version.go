// +kubebuilder:object:generate=true
// +groupName=compute.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "compute.v1.secapi.cloud"
	Version = "v1"

	InstanceSkuResource = "skus"
	InstanceSkuKind     = "InstanceSku"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	InstanceSkuGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: InstanceSkuResource,
	}
	InstanceSkuGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: InstanceSkuKind,
	}
)
