package domain

import "fmt"

// collectionType maps a resource kind to its API collection path segment
// per #216 / #239 (plural form used in metadata.resource and metadata.ref).
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
func collectionType(kind string) string {
	switch kind {
	case "instance":
		return "instances"
	case "block-storage":
		return "block-storages"
	case "image":
		return "images"
	case "network":
		return "networks"
	case "nic":
		return "nics"
	case "public-ip":
		return "public-ips"
	case "internet-gateway":
		return "internet-gateways"
	case "security-group":
		return "security-groups"
	case "security-group-rule":
		return "security-group-rules"
	case "subnet":
		return "subnets"
	case "routing-table", "route-table":
		return "route-tables"
	case "workspace":
		return "workspaces"
	case "instance-sku", "storage-sku", "network-sku":
		return "skus"
	case "role":
		return "roles"
	case "role-assignment":
		return "role-assignments"
	case "region":
		return "regions"
	default:
		// Fallback: append "s" (legacy ResourceFormat behaviour).
		if kind != "" && kind[len(kind)-1] != 's' {
			return kind + "s"
		}
		return kind
	}
}

// FormatResource builds metadata.resource as "{collection}/{name}".
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
// Schema: https://github.com/eu-sovereign-cloud/spec/blob/main/spec/schemas/resource.yaml#L235-L248
func FormatResource[T ~string](resourceType T, name string) string {
	return fmt.Sprintf("%s/%s", collectionType(string(resourceType)), name)
}

// FormatRegionalResource builds metadata.resource as "{collection}/{name}".
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
// Schema: https://github.com/eu-sovereign-cloud/spec/blob/main/spec/schemas/resource.yaml#L235-L248
func FormatRegionalResource[T ~string](resourceType T, name string) string {
	return fmt.Sprintf("%s/%s", collectionType(string(resourceType)), name)
}

// FormatRegionalNetworkScopedResource builds metadata.resource for nested
// network-scoped resources: "networks/{network}/{collection}/{name}".
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
// Schema: https://github.com/eu-sovereign-cloud/spec/blob/main/spec/schemas/resource.yaml#L235-L248
func FormatRegionalNetworkScopedResource[T ~string](network string, resourceType T, name string) string {
	return fmt.Sprintf("networks/%s/%s/%s", network, collectionType(string(resourceType)), name)
}

// FormatResourceRef builds metadata.ref as "{provider}/{collection}/{name}".
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
// Schema: https://github.com/eu-sovereign-cloud/spec/blob/main/spec/schemas/resource.yaml#L146-L186
func FormatResourceRef[T ~string](provider string, resourceType T, name string) string {
	return fmt.Sprintf("%s/%s/%s", provider, collectionType(string(resourceType)), name)
}

// FormatTenantScopedRef builds metadata.ref as
// "{provider}/tenants/{tenant}/{collection}/{name}".
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
// Schema: https://github.com/eu-sovereign-cloud/spec/blob/main/spec/schemas/resource.yaml#L146-L186
func FormatTenantScopedRef[T ~string](provider, tenant string, resourceType T, name string) string {
	return fmt.Sprintf("%s/tenants/%s/%s/%s", provider, tenant, collectionType(string(resourceType)), name)
}

// FormatRegionalWorkspaceScopedRef builds metadata.ref as
// "{provider}/tenants/{tenant}/workspaces/{workspace}/{collection}/{name}".
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
// Schema: https://github.com/eu-sovereign-cloud/spec/blob/main/spec/schemas/resource.yaml#L146-L186
func FormatRegionalWorkspaceScopedRef[T ~string](provider, tenant, workspace string, resourceType T, name string) string {
	return fmt.Sprintf("%s/tenants/%s/workspaces/%s/%s/%s", provider, tenant, workspace, collectionType(string(resourceType)), name)
}

// FormatRegionalTenantScopedRef builds metadata.ref as
// "{provider}/tenants/{tenant}/{collection}/{name}".
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
// Schema: https://github.com/eu-sovereign-cloud/spec/blob/main/spec/schemas/resource.yaml#L146-L186
func FormatRegionalTenantScopedRef[T ~string](provider, tenant string, resourceType T, name string) string {
	return fmt.Sprintf("%s/tenants/%s/%s/%s", provider, tenant, collectionType(string(resourceType)), name)
}

// FormatRegionalNetworkScopedRef builds metadata.ref as
// "{provider}/tenants/{tenant}/workspaces/{workspace}/networks/{network}/{collection}/{name}".
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
// Schema: https://github.com/eu-sovereign-cloud/spec/blob/main/spec/schemas/resource.yaml#L146-L186
func FormatRegionalNetworkScopedRef[T ~string](provider, tenant, workspace, network string, resourceType T, name string) string {
	return fmt.Sprintf("%s/tenants/%s/workspaces/%s/networks/%s/%s/%s",
		provider, tenant, workspace, network, collectionType(string(resourceType)), name)
}
