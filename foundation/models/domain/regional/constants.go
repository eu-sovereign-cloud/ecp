package regional

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
)
