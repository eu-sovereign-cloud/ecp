// +kubebuilder:object:generate=true
// +groupName=authorization.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/global/authorization"
)

const (
	RoleAssignmentResource = "role-assignments"
	RoleAssignmentKind     = "RoleAssignment"
)

var (
	RoleAssignmentGVR = schema.GroupVersionResource{
		Group: authorization.Group, Version: authorization.Version, Resource: RoleAssignmentResource,
	}
	RoleAssignmentGVK = schema.GroupVersionKind{
		Group: authorization.Group, Version: authorization.Version, Kind: RoleAssignmentKind,
	}
)
