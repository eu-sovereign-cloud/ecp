package regionalhandler

import (
	"log/slog"
	"net/http"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// Compute handles HTTP requests for compute resources.
type Compute struct {
	logger *slog.Logger
}

func (c Compute) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkcompute.ListSkusParams) {
	// TODO implement me
	c.logger.Debug("implement me")
}

func (c Compute) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	c.logger.Debug("implement me")
}

func (c Compute) ListInstances(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdkcompute.ListInstancesParams) {
	// TODO implement me
	c.logger.Debug("implement me")
}

func (c Compute) DeleteInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.DeleteInstanceParams) {
	// TODO implement me
	c.logger.Debug("implement me")
}

func (c Compute) GetInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	c.logger.Debug("implement me")
}

func (c Compute) CreateOrUpdateInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.CreateOrUpdateInstanceParams) {
	// TODO implement me
	c.logger.Debug("implement me")
}

func (c Compute) RestartInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.RestartInstanceParams) {
	// TODO implement me
	c.logger.Debug("implement me")
}

func (c Compute) StartInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.StartInstanceParams) {
	// TODO implement me
	c.logger.Debug("implement me")
}

func (c Compute) StopInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.StopInstanceParams) {
	// TODO implement me
	c.logger.Debug("implement me")
}
