package regional

import "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"

type Metadata struct {
	model.CommonMetadata

	Labels      map[string]string
	Annotations map[string]string
	Extensions  map[string]string
	Region      string
	Tenant      string
	Workspace   string
}

func (r *Metadata) GetTenant() string             { return r.Tenant }
func (r *Metadata) GetWorkspace() string          { return r.Workspace }
func (r *Metadata) SetTenant(tenant string)       { r.Tenant = tenant }
func (r *Metadata) SetWorkspace(workspace string) { r.Workspace = workspace }
