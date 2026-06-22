// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "storage.v1.secapi.cloud"
	Version = "v1"

	StorageSKUResource = "skus"
	StorageSKUKind     = "SKU"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	StorageSKUGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: StorageSKUResource,
	}
	StorageSKUGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: StorageSKUKind,
	}
)
