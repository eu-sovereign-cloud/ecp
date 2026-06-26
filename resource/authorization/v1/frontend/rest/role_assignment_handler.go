package rest

import (
	"net/http"
	"strconv"

	sdkauth "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
)

// ListRoleAssignments handles GET /v1/tenants/{tenant}/role-assignments.
func (h *Handler) ListRoleAssignments(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkauth.ListRoleAssignmentsParams) {
	logger := h.Logger.With("provider", "authorization", "resource", "role-assignment")
	frest.HandleList(w, r, logger, ListRoleAssignmentsParamsFromAPI(params, tenant), frest.ListerFromRepo(h.RoleAssignmentReader), RoleAssignmentIteratorToAPI)
}

// DeleteRoleAssignment handles DELETE /v1/tenants/{tenant}/role-assignments/{name}.
func (h *Handler) DeleteRoleAssignment(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkauth.DeleteRoleAssignmentParams) {
	logger := h.Logger.With("provider", "authorization", "resource", "role-assignment", "name", name)
	id := &RoleAssignmentIdentity{name: name, tenant: tenant}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.RoleAssignmentWriter, newRoleAssignmentWithIdentity))
}

// GetRoleAssignment handles GET /v1/tenants/{tenant}/role-assignments/{name}.
func (h *Handler) GetRoleAssignment(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "authorization", "resource", "role-assignment", "name", name)
	id := &RoleAssignmentIdentity{name: name, tenant: tenant}
	frest.HandleGet(w, r, logger, id, frest.GetterFromRepo(h.RoleAssignmentReader, newRoleAssignmentWithIdentity), RoleAssignmentToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateRoleAssignment handles PUT /v1/tenants/{tenant}/role-assignments/{name}.
func (h *Handler) CreateOrUpdateRoleAssignment(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkauth.CreateOrUpdateRoleAssignmentParams) {
	logger := h.Logger.With("provider", "authorization", "resource", "role-assignment", "name", name)
	id := &RoleAssignmentIdentity{name: name, tenant: tenant}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.RoleAssignment, *radom.RoleAssignment, *sdkschema.RoleAssignment]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.RoleAssignmentWriter),
		Updater: frest.UpdaterFromRepo(h.RoleAssignmentWriter),
		APIToDomain: func(sdk sdkschema.RoleAssignment, p persistencepkg.IdentifiableResource) *radom.RoleAssignment {
			return RoleAssignmentFromAPI(sdk, p.(*RoleAssignmentIdentity))
		},
		DomainToAPI: RoleAssignmentToAPIWithVerb(http.MethodPut),
	})
}

// newRoleAssignmentWithIdentity returns a *radom.RoleAssignment populated with identity fields from ir.
func newRoleAssignmentWithIdentity(ir persistencepkg.IdentifiableResource) *radom.RoleAssignment {
	ra := &radom.RoleAssignment{}
	ra.Name = ir.GetName()
	ra.Tenant = ir.GetTenant()
	ra.ResourceVersion = ir.GetVersion()
	return ra
}
