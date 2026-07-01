package rest

import (
	"net/http"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// ListSkus returns HTTP 501: compute SKU listing is not yet implemented.
func (h *Handler) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkcompute.ListSkusParams) {
	h.Logger.DebugContext(r.Context(), "ListSkus not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// GetSku returns HTTP 501: compute SKU retrieval is not yet implemented.
func (h *Handler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetSku not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}
