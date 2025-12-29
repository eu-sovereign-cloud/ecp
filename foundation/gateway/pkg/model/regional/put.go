package regional

type PutParams struct {
	Tenant    string
	Workspace string
	Name      string
}

func (r *PutParams) GetTenant() string             { return r.Tenant }
func (r *PutParams) GetWorkspace() string          { return r.Workspace }
func (r *PutParams) SetTenant(tenant string)       { r.Tenant = tenant }
func (r *PutParams) SetWorkspace(workspace string) { r.Workspace = workspace }
func (r *PutParams) GetName() string               { return r.Name }
func (r *PutParams) SetName(name string)           { r.Name = name }
