package regionalhandler

import (
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/regional/workspace"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/service/handler"
	apiworkspace "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/workspace"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

type Workspace struct {
	Logger *slog.Logger
	List   *workspace.ListWorkspaces
	Get    *workspace.GetWorkspace
	Create *workspace.CreateWorkspace
	Update *workspace.UpdateWorkspace
	Delete *workspace.DeleteWorkspace
}

var _ sdkworkspace.ServerInterface = (*Workspace)(nil) // Ensure Workspace implements the workspace.WorkspaceService interface.

func (h Workspace) ListWorkspaces(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkworkspace.ListWorkspacesParams) {
	handler.HandleList(
		w, r, h.Logger.With("provider", "workspace").With("resource", "workspace"), apiworkspace.ListParamsFromAPI(params, tenant),
		h.List, apiworkspace.DomainToAPIIterator,
	)
}

func (h Workspace) DeleteWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkworkspace.DeleteWorkspaceParams) {
	// TODO implement me
	panic("implement me")
}

func (h Workspace) GetWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	handler.HandleGet(
		w, r, h.Logger.With("provider", "workspace").With("resource", "workspace"), &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: name,
			},
			Tenant: tenant,
		},
		h.Get, apiworkspace.DomainToAPI,
	)
}

func (h Workspace) CreateOrUpdateWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkworkspace.CreateOrUpdateWorkspaceParams) {
	// TODO implement me
	panic("implement me")
}
