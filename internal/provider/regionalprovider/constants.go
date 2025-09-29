package regionalprovider

const (
	// tenantID/resourceName (e.g. "my-tenant/my-resource")
	tenantWideResourceNamePattern = "%s.%s"

	// tenantID/workspaceID/resourceName (e.g. "my-tenant/my-workspace/my-resource")
	workspaceWideResourceNamePattern = "%s.%s.%s"

	tenantLabelKey    = "secapi.cloud/tenant-id"
	workspaceLabelKey = "secapi.cloud/workspace-id"
)
