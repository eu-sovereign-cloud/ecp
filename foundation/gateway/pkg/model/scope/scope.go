package scope

type Scope struct {
	Tenant    string
	Workspace string
}

func (r *Scope) GetTenant() string             { return r.Tenant }
func (r *Scope) GetWorkspace() string          { return r.Workspace }
func (r *Scope) SetTenant(tenant string)       { r.Tenant = tenant }
func (r *Scope) SetWorkspace(workspace string) { r.Workspace = workspace }
