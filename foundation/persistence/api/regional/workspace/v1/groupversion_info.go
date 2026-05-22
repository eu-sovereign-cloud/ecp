// +kubebuilder:object:generate=true
// +groupName=workspace.v1.secapi.cloud
// +versionName=v1

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	// Group is the group name used in this package
	Group = "workspace.v1.secapi.cloud"
	// Version is the version of the API
	Version = "v1"
	// Resource is the resource name for workspaces
	Resource = "workspaces"
	// Kind is the resource kind for workspaces
	Kind = "Workspace"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	WorkspaceGVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: Resource}
	WorkspaceGVK = schema.GroupVersionKind{Group: Group, Version: Version, Kind: Kind}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&Workspace{}, &WorkspaceList{})
}
