package domain

const (
	// WorkspaceScopedResourceFormat defines the format for workspace-scoped resources.
	// Example: tenants/{tenant}/workspaces/{workspace}/{collection}/{name}
	WorkspaceScopedResourceFormat = "tenants/%s/workspaces/%s/%s/%s"

	// TenantScopedResourceFormat defines the format for tenant-scoped resources.
	// Example: tenants/{tenant}/{collection}/{name}
	TenantScopedResourceFormat = "tenants/%s/%s/%s"

	// ResourceFormat defines the short-form path for resources.
	// Example: {collection}/{name}
	ResourceFormat = "%s/%s"

	// RegionalWorkspaceScopedResourceFormat defines the format for regional workspace-scoped resource refs.
	// Example: tenants/{tenant}/workspaces/{workspace}/{collection}/{name}
	RegionalWorkspaceScopedResourceFormat = "tenants/%s/workspaces/%s/%s/%s"

	// RegionalTenantScopedResourceFormat defines the format for regional tenant-scoped resource refs.
	// Example: tenants/{tenant}/{collection}/{name}
	RegionalTenantScopedResourceFormat = "tenants/%s/%s/%s"

	// RegionalResourceFormat defines the short-form path for regional resources.
	// Example: {collection}/{name}
	RegionalResourceFormat = "%s/%s"

	// RegionalNetworkScopedResourceFormat defines the format for network-scoped resource refs.
	// Example: tenants/{tenant}/workspaces/{workspace}/networks/{network}/{collection}/{name}
	RegionalNetworkScopedResourceFormat = "tenants/%s/workspaces/%s/networks/%s/%s/%s"
)
