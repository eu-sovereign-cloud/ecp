package resource

// Identity is a canonical value type that identifies a resource by name and
// version within a Scope. It implements the persistence port's
// IdentifiableResource interface and is used to carry request-side lookup
// coordinates (from path parameters) into read and delete operations without
// fabricating a partially-populated domain model.
//
// Global resources leave Scope zero; tenant-scoped resources set Scope.Tenant;
// workspace-scoped resources set both Scope.Tenant and Scope.Workspace.
type Identity struct {
	Scope

	Name    string
	Version string
}

func (i *Identity) GetName() string    { return i.Name }
func (i *Identity) GetVersion() string { return i.Version }
