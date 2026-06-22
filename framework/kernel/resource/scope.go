package resource

// Scope defines the scope of a resource with tenant and workspace.
// It implements the persistence port's Scope interface.
type Scope struct {
	Tenant    string
	Workspace string
}

func (r *Scope) GetTenant() string    { return r.Tenant }
func (r *Scope) GetWorkspace() string { return r.Workspace }
