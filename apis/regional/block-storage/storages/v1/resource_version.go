// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	storage "github.com/eu-sovereign-cloud/ecp/apis/regional/block-storage"
)

// StorageResource is the resource name for storage SKUs
const StorageResource = "storage"

var (
	StorageGR  = schema.GroupResource{Group: storage.Group, Resource: StorageResource}
	StorageGVR = schema.GroupVersionResource{
		Group: storage.Group, Version: storage.Version, Resource: StorageResource,
	}
)
