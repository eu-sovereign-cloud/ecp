package resource

// Identity is a canonical value type that identifies a resource by name and
// version within a Scope. Both Identity and *Identity implement the persistence
// port's IdentifiableResource interface. It is used to carry request-side
// lookup coordinates (from path parameters) into read and delete operations
// without fabricating a partially-populated domain model.
//
// Global resources leave Scope zero; tenant-scoped resources set Scope.Tenant;
// workspace-scoped resources set both Scope.Tenant and Scope.Workspace.
type Identity struct {
	Scope

	Name    string
	Version string
}

// GetName and GetVersion use value receivers, matching the value-receiver
// GetTenant/GetWorkspace promoted from the embedded Scope. This keeps all four
// methods in the method set of the Identity value — not just *Identity — so the
// value form also satisfies IdentifiableResource. See identity_test.go for the
// compile-time conformance guard.
func (i Identity) GetName() string    { return i.Name }
func (i Identity) GetVersion() string { return i.Version }
