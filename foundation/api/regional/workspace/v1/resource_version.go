// +kubebuilder:object:generate=true
// +groupName=workspace.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace"
)

// WorkspaceResource is the resource name for workspaces.
const (
	WorkspaceResource = "workspaces"
	WorkspaceKind     = "Workspace"
)

var (
	WorkspaceGVR = schema.GroupVersionResource{
		Group: workspace.Group, Version: workspace.Version, Resource: WorkspaceResource,
	}
	WorkspaceGVK = schema.GroupVersionKind{
		Group: workspace.Group, Version: workspace.Version, Kind: WorkspaceKind,
	}
)
