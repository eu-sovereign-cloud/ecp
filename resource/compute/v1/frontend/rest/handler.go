// Package rest provides REST↔domain conversion and HTTP handlers for the compute API group.
package rest

import (
	"log/slog"
	"net/http"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// Handler is the HTTP handler for the compute API group.
// Instance CRUD methods are in instance_handler.go.
// SKU methods and instance power-state operations (start/stop/restart) are not yet
// implemented and return 501 (stubs below).
type Handler struct {
	InstanceReader persistencepkg.ReaderRepo[*instancedom.Instance]
	InstanceWriter persistencepkg.WriterRepo[*instancedom.Instance]
	Logger         *slog.Logger
}

var _ sdkcompute.ServerInterface = (*Handler)(nil)

// ListSkus is not yet implemented.
func (h *Handler) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkcompute.ListSkusParams) {
	h.Logger.DebugContext(r.Context(), "ListSkus not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// GetSku is not yet implemented.
func (h *Handler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetSku not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// RestartInstance is not yet implemented.
func (h *Handler) RestartInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.RestartInstanceParams) {
	h.Logger.DebugContext(r.Context(), "RestartInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// StartInstance is not yet implemented.
func (h *Handler) StartInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.StartInstanceParams) {
	h.Logger.DebugContext(r.Context(), "StartInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// StopInstance is not yet implemented.
func (h *Handler) StopInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.StopInstanceParams) {
	h.Logger.DebugContext(r.Context(), "StopInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}
