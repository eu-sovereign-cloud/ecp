package rest

import (
	"log/slog"
	"net/http"
	"strconv"

	sdkauth "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role/v1"
)

// Handler is the HTTP handler for the authorization API group.
// It implements the full sdkauth.ServerInterface: role methods directly, and
// role-assignment methods as HTTP 501 stubs (deferred to a later implementation).
type Handler struct {
	Reader persistencepkg.ReaderRepo[*roledom.Role]
	Writer persistencepkg.WriterRepo[*roledom.Role]
	Logger *slog.Logger
}

var _ sdkauth.ServerInterface = (*Handler)(nil)

// ListRoles handles GET /v1/tenants/{tenant}/roles.
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkauth.ListRolesParams) {
	logger := h.Logger.With("provider", "authorization", "resource", "role")
	frest.HandleList(w, r, logger, ListParamsFromAPI(params, tenant), frest.ListerFromRepo(h.Reader), RoleIteratorToAPI)
}

// DeleteRole handles DELETE /v1/tenants/{tenant}/roles/{name}.
func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkauth.DeleteRoleParams) {
	logger := h.Logger.With("provider", "authorization", "resource", "role", "name", name)
	id := &RoleIdentity{name: name, tenant: tenant}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.Writer, newRoleWithIdentity))
}

// GetRole handles GET /v1/tenants/{tenant}/roles/{name}.
func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "authorization", "resource", "role", "name", name)
	id := &RoleIdentity{name: name, tenant: tenant}
	frest.HandleGet(w, r, logger, id, frest.GetterFromRepo(h.Reader, newRoleWithIdentity), RoleToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateRole handles PUT /v1/tenants/{tenant}/roles/{name}.
func (h *Handler) CreateOrUpdateRole(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkauth.CreateOrUpdateRoleParams) {
	logger := h.Logger.With("provider", "authorization", "resource", "role", "name", name)
	id := &RoleIdentity{name: name, tenant: tenant}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Role, *roledom.Role, *sdkschema.Role]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.Writer),
		Updater: frest.UpdaterFromRepo(h.Writer),
		APIToDomain: func(sdk sdkschema.Role, p persistencepkg.IdentifiableResource) *roledom.Role {
			return RoleFromAPI(sdk, p.(*RoleIdentity))
		},
		DomainToAPI: RoleToAPIWithVerb(http.MethodPut),
	})
}

// ListRoleAssignments is a stub — role-assignment vertical is not yet implemented.
func (h *Handler) ListRoleAssignments(w http.ResponseWriter, r *http.Request, _ sdkschema.TenantPathParam, _ sdkauth.ListRoleAssignmentsParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// DeleteRoleAssignment is a stub — role-assignment vertical is not yet implemented.
func (h *Handler) DeleteRoleAssignment(w http.ResponseWriter, r *http.Request, _ sdkschema.TenantPathParam, _ sdkschema.ResourcePathParam, _ sdkauth.DeleteRoleAssignmentParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// GetRoleAssignment is a stub — role-assignment vertical is not yet implemented.
func (h *Handler) GetRoleAssignment(w http.ResponseWriter, r *http.Request, _ sdkschema.TenantPathParam, _ sdkschema.ResourcePathParam) {
	w.WriteHeader(http.StatusNotImplemented)
}

// CreateOrUpdateRoleAssignment is a stub — role-assignment vertical is not yet implemented.
func (h *Handler) CreateOrUpdateRoleAssignment(w http.ResponseWriter, r *http.Request, _ sdkschema.TenantPathParam, _ sdkschema.ResourcePathParam, _ sdkauth.CreateOrUpdateRoleAssignmentParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// newRoleWithIdentity returns a *roledom.Role populated with identity fields from ir.
func newRoleWithIdentity(ir persistencepkg.IdentifiableResource) *roledom.Role {
	r := &roledom.Role{}
	r.Name = ir.GetName()
	r.Tenant = ir.GetTenant()
	r.ResourceVersion = ir.GetVersion()
	return r
}
