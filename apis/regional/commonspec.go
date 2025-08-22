package regional

// CommonSpec defines the common fields for regional resources.
type CommonSpec struct {
	// Tenant is the identifier for the tenant that owns this resource.
	Tenant string `json:"tenant"`

	// Workspace is the identifier for the workspace this resource belongs to.
	Workspace string `json:"workspace"`

	// Region is the identifier for the region where this resource is located.
	Region string `json:"region"`

	// Zone is the identifier for the availability zone within the region.
	// This field is optional.
	// +optional
	Zone string `json:"zone,omitempty"`

	// Name is the user-defined name of the resource.
	Name string `json:"name"`
}
