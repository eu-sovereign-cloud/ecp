package rest

import (
	"net/http"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// ListInstances returns HTTP 501: compute instance listing is not yet implemented.
func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdkcompute.ListInstancesParams) {
	h.Logger.DebugContext(r.Context(), "ListInstances not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// DeleteInstance returns HTTP 501: compute instance deletion is not yet implemented.
func (h *Handler) DeleteInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.DeleteInstanceParams) {
	h.Logger.DebugContext(r.Context(), "DeleteInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// GetInstance returns HTTP 501: compute instance retrieval is not yet implemented.
func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// CreateOrUpdateInstance returns HTTP 501: compute instance creation/update is not yet implemented.
func (h *Handler) CreateOrUpdateInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.CreateOrUpdateInstanceParams) {
	h.Logger.DebugContext(r.Context(), "CreateOrUpdateInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// RestartInstance returns HTTP 501: compute instance restart is not yet implemented.
func (h *Handler) RestartInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.RestartInstanceParams) {
	h.Logger.DebugContext(r.Context(), "RestartInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// StartInstance returns HTTP 501: compute instance start is not yet implemented.
func (h *Handler) StartInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.StartInstanceParams) {
	h.Logger.DebugContext(r.Context(), "StartInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// StopInstance returns HTTP 501: compute instance stop is not yet implemented.
func (h *Handler) StopInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.StopInstanceParams) {
	h.Logger.DebugContext(r.Context(), "StopInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}
