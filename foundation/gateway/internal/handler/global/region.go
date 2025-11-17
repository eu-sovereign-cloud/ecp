package globalhandler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"k8s.io/apimachinery/pkg/api/errors"

	globalprovider "github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/adapter/global"
)

// Region handles HTTP requests for region data.
type Region struct {
	provider globalprovider.RegionProvider
	logger   *slog.Logger
}

// NewRegion creates a new handler for region endpoints.
func NewRegion(logger *slog.Logger, p globalprovider.RegionProvider) *Region {
	return &Region{provider: p, logger: logger.With("component", "Region")}
}

var _ regionv1.ServerInterface = (*Region)(nil)

// ListRegions handles requests to list all available regions.
func (h *Region) ListRegions(w http.ResponseWriter, r *http.Request, params regionv1.ListRegionsParams) {
	iterator, err := h.provider.ListRegions(r.Context(), params)
	if err != nil {
		h.logger.Error("failed to list regions", "error", err)
		http.Error(w, "failed to list regions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", string(schema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(iterator)
	if err != nil {
		h.logger.Error("failed to encode regions", "error", err)
		http.Error(w, "failed to encode regions: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetRegion handles requests to get a specific region by name.
func (h *Region) GetRegion(w http.ResponseWriter, r *http.Request, name schema.ResourcePathParam) {
	reg, err := h.provider.GetRegion(r.Context(), name)
	if err != nil {
		if errors.IsNotFound(err) {
			h.logger.InfoContext(r.Context(), "region not found", slog.String("region", name))
			http.Error(w, fmt.Sprintf("region (%s) not found", name), http.StatusNotFound)
			return
		}

		// For all other errors (e.g., connection issues, CRD not registered),
		// log the error and return a 500 Internal Server Error.
		h.logger.ErrorContext(r.Context(), "failed to get region", slog.String("region", name), slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", string(schema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(reg)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to encode region", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
