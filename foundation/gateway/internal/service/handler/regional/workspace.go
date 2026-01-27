package regionalhandler

import (
	"log/slog"
	"net/http"
	"strconv"

	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/regional/workspace"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/service/handler"
	apiworkspace "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/workspace"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

type Workspace struct {
	Logger *slog.Logger
	Create *workspace.CreateWorkspace
	Update *workspace.UpdateWorkspace
	Delete *workspace.DeleteWorkspace
	List   *workspace.ListWorkspace
	Get    *workspace.GetWorkspace
}

var _ sdkworkspace.ServerInterface = (*Workspace)(nil)

func (h Workspace) ListWorkspaces(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, params sdkworkspace.ListWorkspacesParams) {
	handler.HandleList(
		w, r, h.Logger.With("provider", "workspace").With("resource", "workspace"),
		apiworkspace.ListParamsFromAPI(params, tenant), h.List, apiworkspace.DomainToAPIIterator,
	)
}

func (h Workspace) DeleteWorkspace(
	w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, name schema.ResourcePathParam, params sdkworkspace.DeleteWorkspaceParams,
) {
	var resourceVersion string
	if params.IfUnmodifiedSince != nil {
		resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	ir := &regional.Metadata{
		Scope: scope.Scope{
			Tenant:    tenant,
			Workspace: name,
		},
		CommonMetadata: model.CommonMetadata{
			Name:            name,
			ResourceVersion: resourceVersion,
		},
	}

	handler.HandleDelete(w, r, h.Logger.With("provider", "workspace").With("resource", "workspace"), ir, h.Delete)
}

func (h Workspace) GetWorkspace(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, name schema.ResourcePathParam) {
	ir := &regional.Metadata{
		Scope: scope.Scope{
			Tenant:    tenant,
			Workspace: name,
		},
		CommonMetadata: model.CommonMetadata{
			Name: name,
		},
	}

	handler.HandleGet(w, r, h.Logger.With("provider", "workspace").With("resource", "workspace"), ir, h.Get, apiworkspace.DomainToAPI)
}

func (h Workspace) CreateOrUpdateWorkspace(
	w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, name schema.ResourcePathParam, params sdkworkspace.CreateOrUpdateWorkspaceParams,
) {
	var resourceVersion string
	if params.IfUnmodifiedSince != nil {
		resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}

	upsertOptions := handler.UpsertOptions[schema.Workspace, *regional.WorkspaceDomain, schema.Workspace]{
		Params: &regional.Metadata{
			Scope: scope.Scope{
				Tenant:    tenant,
				Workspace: name,
			},
			CommonMetadata: model.CommonMetadata{
				Name:            name,
				ResourceVersion: resourceVersion,
			},
		},
		Creator:     h.Create,
		Updater:     h.Update,
		SDKToDomain: apiworkspace.APIToDomain,
		DomainToSDK: apiworkspace.DomainToAPI,
	}

	handler.HandleUpsert(
		w, r, h.Logger.With("provider", "workspace").With("resource", "workspace"), upsertOptions,
	)
}
