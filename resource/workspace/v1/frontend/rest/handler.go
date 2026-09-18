package rest

import (
	"log/slog"
	"net/http"
	"strconv"

	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// Handler is the HTTP handler for workspace resources.
// It implements the full sdkworkspace.ServerInterface.
type Handler struct {
	Reader persistencepkg.ReaderRepo[*wsdom.Workspace]
	Writer persistencepkg.WriterRepo[*wsdom.Workspace]
	Logger *slog.Logger
	// Region is the default region this handler serves: what a request that named no
	// region is served as. A gateway may serve several, in which case the region resolved
	// for the request (resource.RegionFromContext) wins. Empty on the global server.
	Region string
}

var _ sdkworkspace.ServerInterface = (*Handler)(nil)

// ListWorkspaces handles GET /v1/tenants/{tenant}/workspaces.
func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkworkspace.ListWorkspacesParams) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace")
	region := resource.RegionFromContext(r.Context(), h.Region)
	frest.HandleList(w, r, logger, listParamsFromAPI(params, tenant, region), frest.ListerFromRepo(h.Reader), workspaceIteratorToAPI)
}

// DeleteWorkspace handles DELETE /v1/tenants/{tenant}/workspaces/{name}.
func (h *Handler) DeleteWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkworkspace.DeleteWorkspaceParams) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace")
	id := h.identity(r, tenant, name)
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.Writer, newWorkspaceWithIdentity))
}

// GetWorkspace handles GET /v1/tenants/{tenant}/workspaces/{name}.
func (h *Handler) GetWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace")
	ir := h.identity(r, tenant, name)
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.Reader, newWorkspaceWithIdentity), workspaceToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateWorkspace handles PUT /v1/tenants/{tenant}/workspaces/{name}.
func (h *Handler) CreateOrUpdateWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkworkspace.CreateOrUpdateWorkspaceParams) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace")
	id := h.identity(r, tenant, name)
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Workspace, *wsdom.Workspace, *sdkschema.Workspace]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.Writer),
		Updater: frest.UpdaterFromRepo(h.Writer),
		APIToDomain: func(sdk sdkschema.Workspace, p persistencepkg.IdentifiableResource) (*wsdom.Workspace, error) {
			return workspaceFromAPI(sdk, p.(*resource.Identity), id.Region)
		},
		DomainToAPI: workspaceToAPIWithVerb(http.MethodPut),
	})
}

// identity builds the lookup key for a single workspace. A workspace is identified by tenant,
// name *and* region: the same name in the same tenant is a different workspace in a different
// region, so the region the request was addressed to belongs in the key rather than only on the
// created resource.
func (h *Handler) identity(r *http.Request, tenant, name string) *resource.Identity {
	return &resource.Identity{
		Name:   name,
		Scope:  resource.Scope{Tenant: tenant},
		Region: resource.RegionFromContext(r.Context(), h.Region),
	}
}

// newWorkspaceWithIdentity returns a *wsdom.Workspace populated with identity fields from ir.
func newWorkspaceWithIdentity(ir persistencepkg.IdentifiableResource) *wsdom.Workspace {
	d := &wsdom.Workspace{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.ResourceVersion = ir.GetVersion()
	if regionScope, ok := ir.(persistencepkg.RegionScope); ok {
		d.Region = regionScope.GetRegion()
	}
	return d
}
