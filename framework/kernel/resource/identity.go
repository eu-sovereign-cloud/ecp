package resource

// Identity is a canonical value type that identifies a resource by name and
// version within a Scope. Both Identity and *Identity implement the persistence
// port's IdentifiableResource interface. It is used to carry request-side
// lookup coordinates (from path parameters) into read and delete operations
// without fabricating a partially-populated domain model.
//
// Global resources leave Scope zero; tenant-scoped resources set Scope.Tenant;
// workspace-scoped resources set both Scope.Tenant and Scope.Workspace.
//
// Region qualifies the identity for a resource whose backend keys it by region as well as
// by Scope (Workspace — see resource/workspace/v1). Every other resource leaves it zero, and
// a zero Region resolves exactly as before.
type Identity struct {
	Scope

	Name    string
	Version string
	Region  string
}

// GetName, GetVersion and GetRegion use value receivers, matching the value-receiver
// GetTenant/GetWorkspace promoted from the embedded Scope. This keeps all five
// methods in the method set of the Identity value — not just *Identity — so the
// value form also satisfies IdentifiableResource. See identity_test.go for the
// compile-time conformance guard.
func (i Identity) GetName() string    { return i.Name }
func (i Identity) GetVersion() string { return i.Version }
func (i Identity) GetRegion() string  { return i.Region }
