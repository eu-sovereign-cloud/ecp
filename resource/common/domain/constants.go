package domain

const (
	// WorkspaceScopedResourceFormat defines the format for workspace-scoped resources. It pluralizes the resource type.
	// Example: tenant/{tenant_name}/workspace/{workspace_name}/{resource_type}s/{resource_name}
	WorkspaceScopedResourceFormat = "tenants/%s/workspaces/%s/%ss/%s"

	// TenantScopedResourceFormat defines the format for tenant-scoped resources. It pluralizes the resource type.
	// Example: tenant/{tenant_name}/{resource_type}s/{resource_name}
	TenantScopedResourceFormat = "tenants/%s/%ss/%s"

	// ResourceFormat defines the general format for resources. It pluralizes the resource type.
	// Example: {resource_type}s/{resource_name}
	ResourceFormat = "%ss/%s"

	// RegionalWorkspaceScopedResourceFormat defines the format for regional workspace-scoped resources backed by a provider.
	// Example: tenants/{tenant_name}/workspaces/{workspace_name}/providers/{resource_type}/{resource_name}
	RegionalWorkspaceScopedResourceFormat = "tenants/%s/workspaces/%s/providers/%s/%s"

	// RegionalTenantScopedResourceFormat defines the format for regional tenant-scoped resources backed by a provider.
	// Example: tenants/{tenant_name}/providers/{resource_type}/{resource_name}
	RegionalTenantScopedResourceFormat = "tenants/%s/providers/%s/%s"

	// RegionalResourceFormat defines the short-form path for regional resources within a provider namespace.
	// Example: {resource_type}/{resource_name}
	RegionalResourceFormat = "%s/%s"
)
