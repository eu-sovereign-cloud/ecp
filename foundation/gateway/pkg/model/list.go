package model

// ListParams - parameters for listing resources
type ListParams struct {
	Tenant    string
	Workspace string
	Limit     int
	SkipToken string
	Selector  string
}

func (r *ListParams) GetTenant() string             { return r.Tenant }
func (r *ListParams) GetWorkspace() string          { return r.Workspace }
func (r *ListParams) SetTenant(tenant string)       { r.Tenant = tenant }
func (r *ListParams) SetWorkspace(workspace string) { r.Workspace = workspace }
