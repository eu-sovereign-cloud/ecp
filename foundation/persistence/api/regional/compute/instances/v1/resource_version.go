// +kubebuilder:object:generate=true
// +groupName=compute.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/compute"
)

const (
	InstanceResource = "instances"
	InstanceKind     = "Instance"
)

var (
	InstanceGVR = schema.GroupVersionResource{
		Group: compute.Group, Version: compute.Version, Resource: InstanceResource,
	}
	InstanceGVK = schema.GroupVersionKind{
		Group: compute.Group, Version: compute.Version, Kind: InstanceKind,
	}
)
