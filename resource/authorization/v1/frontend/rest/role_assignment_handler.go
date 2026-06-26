package rest

import (
	"net/http"

	sdkauth "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// ListRoleAssignments is a stub — role-assignment endpoint is not yet wired.
func (h *Handler) ListRoleAssignments(w http.ResponseWriter, _ *http.Request, _ sdkschema.TenantPathParam, _ sdkauth.ListRoleAssignmentsParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// DeleteRoleAssignment is a stub — role-assignment endpoint is not yet wired.
func (h *Handler) DeleteRoleAssignment(w http.ResponseWriter, _ *http.Request, _ sdkschema.TenantPathParam, _ sdkschema.ResourcePathParam, _ sdkauth.DeleteRoleAssignmentParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// GetRoleAssignment is a stub — role-assignment endpoint is not yet wired.
func (h *Handler) GetRoleAssignment(w http.ResponseWriter, _ *http.Request, _ sdkschema.TenantPathParam, _ sdkschema.ResourcePathParam) {
	w.WriteHeader(http.StatusNotImplemented)
}

// CreateOrUpdateRoleAssignment is a stub — role-assignment endpoint is not yet wired.
func (h *Handler) CreateOrUpdateRoleAssignment(w http.ResponseWriter, _ *http.Request, _ sdkschema.TenantPathParam, _ sdkschema.ResourcePathParam, _ sdkauth.CreateOrUpdateRoleAssignmentParams) {
	w.WriteHeader(http.StatusNotImplemented)
}
