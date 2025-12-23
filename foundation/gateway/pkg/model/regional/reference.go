package regional

// ReferenceObject is a domain type representing a reference to another resource.
// It uses a structured object format that can reference resources across
// workspaces or regions
type ReferenceObject struct {
	// Provider of the resource. If empty, inferred from context.
	Provider string
	// Region of the resource. If empty, inferred from context.
	Region string
	// Resource is the name and type in format `<type>/<name>`.
	Resource string
	// Tenant of the resource. If empty, inferred from context.
	Tenant string
	// Workspace of the resource. If empty, inferred from context.
	Workspace string
}
