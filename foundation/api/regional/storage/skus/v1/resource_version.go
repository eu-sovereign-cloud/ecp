// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
)

// SKUResource is the resource name for storage SKUs.
const SKUResource = "skus"

var (
	SKUGR  = schema.GroupResource{Group: storage.Group, Resource: SKUResource}
	SKUGVR = schema.GroupVersionResource{
		Group: storage.Group, Version: storage.Version, Resource: SKUResource,
	}
)
