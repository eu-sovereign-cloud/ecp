package globalhandler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/global/region"
)

// Region handles HTTP requests for region data.
type Region struct {
	List   *region.List
	Get    *region.Get
	Logger *slog.Logger
}

var _ regionv1.ServerInterface = (*Region)(nil)

// ListRegions handles requests to list all available regions.
func (h *Region) ListRegions(w http.ResponseWriter, r *http.Request, params regionv1.ListRegionsParams) {
	iterator, err := h.List.Do(r.Context(), params)
	if err != nil {
		h.Logger.Error("failed to list regions", "error", err)
		http.Error(w, "failed to list regions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", string(schema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(iterator)
	if err != nil {
		h.Logger.Error("failed to encode regions", "error", err)
		http.Error(w, "failed to encode regions: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetRegion handles requests to get a specific region by name.
func (h *Region) GetRegion(w http.ResponseWriter, r *http.Request, name schema.ResourcePathParam) {
	reg, err := h.Get.Do(r.Context(), name)
	if err != nil {
		if errors.IsNotFound(err) {
			h.Logger.InfoContext(r.Context(), "region not found", slog.String("region", name))
			http.Error(w, fmt.Sprintf("region (%s) not found", name), http.StatusNotFound)
			return
		}

		// For all other errors (e.g., connection issues, CRD not registered),
		// log the error and return a 500 Internal Server Error.
		h.Logger.ErrorContext(r.Context(), "failed to get region", slog.String("region", name), slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", string(schema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(reg)
	if err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to encode region", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
