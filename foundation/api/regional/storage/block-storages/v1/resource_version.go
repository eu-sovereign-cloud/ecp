// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
)

// BlockStorageResource is the resource name for storage block-storage instances.
const BlockStorageResource = "block-storages"

var (
	BlockStorageGR  = schema.GroupResource{Group: storage.Group, Resource: BlockStorageResource}
	BlockStorageGVR = schema.GroupVersionResource{
		Group: storage.Group, Version: storage.Version, Resource: BlockStorageResource,
	}
)
