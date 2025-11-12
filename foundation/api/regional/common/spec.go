// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

package common

// TenantRegionalCommonSpec defines the common spec fields for regional resources scoped to a tenant.
type TenantRegionalCommonSpec struct {
	// Tenant is the identifier for the tenant that owns this resource.
	Tenant string `json:"tenant"`

	// Region is the identifier for the region where this resource is located.
	Region string `json:"region"`

	// Name is the user-defined name of the resource.
	Name string `json:"name"`
}

// WorkspaceRegionalCommonSpec defines the common spec fields for regional resources scoped to a workspace.
type WorkspaceRegionalCommonSpec struct {
	// Workspace is the identifier for the workspace this resource belongs to.
	Workspace string `json:"workspace"`

	TenantRegionalCommonSpec `json:",inline"`
}
