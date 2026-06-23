package rest

import (
	"log/slog"
	"net/http"
	"strconv"

	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1"
)

// Handler is the HTTP handler for workspace resources.
// It implements the full sdkworkspace.ServerInterface.
type Handler struct {
	Reader persistencepkg.ReaderRepo[*wsdom.Workspace]
	Writer persistencepkg.WriterRepo[*wsdom.Workspace]
	Logger *slog.Logger
}

var _ sdkworkspace.ServerInterface = (*Handler)(nil)

// ListWorkspaces handles GET /v1/tenants/{tenant}/workspaces.
func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkworkspace.ListWorkspacesParams) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace")
	frest.HandleList(w, r, logger, ListParamsFromAPI(params, tenant), frest.ListerFromRepo(h.Reader), DomainToAPIIterator)
}

// DeleteWorkspace handles DELETE /v1/tenants/{tenant}/workspaces/{name}.
func (h *Handler) DeleteWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkworkspace.DeleteWorkspaceParams) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace", "name", name)
	id := &WorkspaceIdentity{name: name, tenant: tenant}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.Writer, newWorkspaceWithIdentity))
}

// GetWorkspace handles GET /v1/tenants/{tenant}/workspaces/{name}.
func (h *Handler) GetWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace", "name", name)
	ir := &WorkspaceIdentity{name: name, tenant: tenant}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.Reader, newWorkspaceWithIdentity), DomainToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateWorkspace handles PUT /v1/tenants/{tenant}/workspaces/{name}.
func (h *Handler) CreateOrUpdateWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkworkspace.CreateOrUpdateWorkspaceParams) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace", "name", name)
	id := &WorkspaceIdentity{name: name, tenant: tenant}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Workspace, *wsdom.Workspace, *sdkschema.Workspace]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.Writer),
		Updater: frest.UpdaterFromRepo(h.Writer),
		APIToDomain: func(sdk sdkschema.Workspace, p persistencepkg.IdentifiableResource) *wsdom.Workspace {
			return APIToDomain(sdk, p.(*WorkspaceIdentity), region)
		},
		DomainToAPI: DomainToAPIWithVerb(http.MethodPut),
	})
}

// newWorkspaceWithIdentity returns a *wsdom.Workspace populated with identity fields from ir.
func newWorkspaceWithIdentity(ir persistencepkg.IdentifiableResource) *wsdom.Workspace {
	d := &wsdom.Workspace{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.ResourceVersion = ir.GetVersion()
	return d
}
