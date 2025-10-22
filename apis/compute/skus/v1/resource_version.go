// +kubebuilder:object:generate=true
// +groupName=compute.v1.secapi.cloud
// +versionName=v1

package v1

import (
	storage "github.com/eu-sovereign-cloud/ecp/apis/compute"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const InstanceSKUResource = "instance-skus"

var (
	InstanceSKUGR  = schema.GroupResource{Group: storage.Group, Resource: InstanceSKUResource}
	InstanceSKUGVR = schema.GroupVersionResource{
		Group: storage.Group, Version: storage.Version, Resource: InstanceSKUResource,
	}
)
