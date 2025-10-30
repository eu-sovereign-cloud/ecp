package regionalhandler

import (
	"net/http"

	compute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/internal/provider/regionalprovider"
)

// ComputeHandler handles HTTP requests for compute resources.
// It uses a ComputeProvider to perform the actual operations.
type ComputeHandler struct {
	provider regionalprovider.ComputeProvider
}

// NewComputeHandler creates a new ComputeHandler with the given provider.
func NewComputeHandler(provider regionalprovider.ComputeProvider) ComputeHandler {
	return ComputeHandler{
		provider: provider,
	}
}

func (c ComputeHandler) ListSkus(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, params compute.ListSkusParams) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeHandler) GetSku(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, name schema.ResourcePathParam) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeHandler) ListInstances(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, params compute.ListInstancesParams) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeHandler) DeleteInstance(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, name schema.ResourcePathParam, params compute.DeleteInstanceParams) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeHandler) GetInstance(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, name schema.ResourcePathParam) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeHandler) CreateOrUpdateInstance(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, name schema.ResourcePathParam, params compute.CreateOrUpdateInstanceParams) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeHandler) RestartInstance(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, name schema.ResourcePathParam, params compute.RestartInstanceParams) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeHandler) StartInstance(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, name schema.ResourcePathParam, params compute.StartInstanceParams) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeHandler) StopInstance(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, name schema.ResourcePathParam, params compute.StopInstanceParams) {
	// TODO implement me
	panic("implement me")
}
