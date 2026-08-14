package domain

// Reference is a domain type representing a reference to another resource.
// It uses a structured object format that can reference resources across
// workspaces or regions.
//
// Scope may be represented by the explicit fields or embedded in Resource.
// Conversions preserve the representation supplied by the client.
type Reference struct {
	// Provider of the resource. If empty, inferred from context.
	Provider string
	// Region of the resource. If empty, inferred from context.
	Region string
	// Resource is the resource-specific path within its workspace context.
	// Flat: {collection}/{name} (e.g. instances/my-server).
	// Nested: networks/{network}/{collection}/{name}
	//   (e.g. networks/my-net/route-tables/my-rt).
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	// Schema: https://github.com/eu-sovereign-cloud/spec/blob/main/spec/schemas/resource.yaml#L235-L248
	Resource string
	// Tenant of the resource. If empty, inferred from context.
	Tenant string
	// Workspace of the resource. If empty, inferred from context.
	Workspace string
}
