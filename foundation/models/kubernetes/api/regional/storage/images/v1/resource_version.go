// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/storage"
)

const (
	ImageResource = "images"
	ImageKind     = "Image"
)

var (
	ImageGVR = schema.GroupVersionResource{
		Group: storage.Group, Version: storage.Version, Resource: ImageResource,
	}
	ImageGVK = schema.GroupVersionKind{
		Group: storage.Group, Version: storage.Version, Kind: ImageKind,
	}
)
