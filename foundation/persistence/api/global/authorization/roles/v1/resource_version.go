// +kubebuilder:object:generate=true
// +groupName=authorization.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/global/authorization"
)

const (
	RoleResource = "roles"
	RoleKind     = "Role"
)

var (
	RoleGVR = schema.GroupVersionResource{
		Group: authorization.Group, Version: authorization.Version, Resource: RoleResource,
	}
	RoleGVK = schema.GroupVersionKind{
		Group: authorization.Group, Version: authorization.Version, Kind: RoleKind,
	}
)
