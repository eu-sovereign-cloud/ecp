package domain

// Reference is a domain type representing a reference to another resource.
// It uses a structured object format that can reference resources across
// workspaces or regions.
//
// The scope may arrive either as its own fields ({tenant: "t", resource: "skus/s"}) or spelled
// out in the path ({resource: "seca.storage/v1/tenants/t/skus/s"}) - the same reference, two
// representations. A reference is stored and echoed back exactly as written; use
// backend.ParseReference to read the pieces.
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
