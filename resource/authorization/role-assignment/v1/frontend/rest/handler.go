package rest

import (
	"log/slog"
	"net/http"
	"strconv"

	sdkauthz "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1"
)

// RoleAssignmentHandler implements the role-assignment methods of the authorization
// group's sdkauthz.ServerInterface. It is intended to be embedded by the authorization
// group owner Handler (not yet created) so the role-assignment methods are promoted onto it.
type RoleAssignmentHandler struct {
	Reader persistencepkg.ReaderRepo[*radom.RoleAssignment]
	Writer persistencepkg.WriterRepo[*radom.RoleAssignment]
	Logger *slog.Logger
}

// ListRoleAssignments handles GET /v1/tenants/{tenant}/role-assignments.
func (h *RoleAssignmentHandler) ListRoleAssignments(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkauthz.ListRoleAssignmentsParams) {
	logger := h.Logger.With("provider", "authorization", "resource", "role-assignment")
	frest.HandleList(w, r, logger, ListParamsFromAPI(params, tenant), frest.ListerFromRepo(h.Reader), RoleAssignmentIteratorToAPI)
}

// DeleteRoleAssignment handles DELETE /v1/tenants/{tenant}/role-assignments/{name}.
func (h *RoleAssignmentHandler) DeleteRoleAssignment(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkauthz.DeleteRoleAssignmentParams) {
	logger := h.Logger.With("provider", "authorization", "resource", "role-assignment", "name", name)
	id := &RoleAssignmentIdentity{name: name, tenant: tenant}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.Writer, newRoleAssignmentWithIdentity))
}

// GetRoleAssignment handles GET /v1/tenants/{tenant}/role-assignments/{name}.
func (h *RoleAssignmentHandler) GetRoleAssignment(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "authorization", "resource", "role-assignment", "name", name)
	ir := &RoleAssignmentIdentity{name: name, tenant: tenant}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.Reader, newRoleAssignmentWithIdentity), RoleAssignmentToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateRoleAssignment handles PUT /v1/tenants/{tenant}/role-assignments/{name}.
func (h *RoleAssignmentHandler) CreateOrUpdateRoleAssignment(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkauthz.CreateOrUpdateRoleAssignmentParams) {
	logger := h.Logger.With("provider", "authorization", "resource", "role-assignment", "name", name)
	id := &RoleAssignmentIdentity{name: name, tenant: tenant}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.RoleAssignment, *radom.RoleAssignment, *sdkschema.RoleAssignment]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.Writer),
		Updater: frest.UpdaterFromRepo(h.Writer),
		APIToDomain: func(sdk sdkschema.RoleAssignment, p persistencepkg.IdentifiableResource) *radom.RoleAssignment {
			return RoleAssignmentFromAPI(sdk, p.(*RoleAssignmentIdentity), region)
		},
		DomainToAPI: RoleAssignmentToAPIWithVerb(http.MethodPut),
	})
}
