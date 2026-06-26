package rest

import (
	"net/http"
	"strconv"

	sdkauth "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
)

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

// newRoleWithIdentity returns a *roledom.Role populated with identity fields from ir.
func newRoleWithIdentity(ir persistencepkg.IdentifiableResource) *roledom.Role {
	r := &roledom.Role{}
	r.Name = ir.GetName()
	r.Tenant = ir.GetTenant()
	r.ResourceVersion = ir.GetVersion()
	return r
}
