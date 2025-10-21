// +kubebuilder:object:generate=true
// +groupName=compute.v1.secapi.cloud
// +versionName=v1

package v1

import (
	storage "github.com/eu-sovereign-cloud/ecp/apis/compute"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const ComputeSKUResource = "compute-skus"

var (
	StorageSKUGR  = schema.GroupResource{Group: storage.Group, Resource: ComputeSKUResource}
	StorageSKUGVR = schema.GroupVersionResource{
		Group: storage.Group, Version: storage.Version, Resource: ComputeSKUResource,
	}
)
