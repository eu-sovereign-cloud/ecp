// +kubebuilder:object:generate=true
// +groupName=workspace.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "workspace.v1.secapi.cloud"
	Version = "v1"

	WorkspaceResource = "workspaces"
	WorkspaceKind     = "Workspace"
)

var (
	GroupVersion  = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	WorkspaceGVR = schema.GroupVersionResource{
		Group: Group, Version: Version, Resource: WorkspaceResource,
	}
	WorkspaceGVK = schema.GroupVersionKind{
		Group: Group, Version: Version, Kind: WorkspaceKind,
	}
)
