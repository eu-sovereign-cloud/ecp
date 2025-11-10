// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package v1

import (
    "k8s.io/apimachinery/pkg/runtime/schema"

    storage "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/block-storage"
)

// StorageSKUResource is the resource name for storage SKUs
const StorageSKUResource = "storage-skus"

var (
	StorageSKUGR  = schema.GroupResource{Group: storage.Group, Resource: StorageSKUResource}
	StorageSKUGVR = schema.GroupVersionResource{
		Group: storage.Group, Version: storage.Version, Resource: StorageSKUResource,
	}
)
