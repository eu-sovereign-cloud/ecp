package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	authv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/eu-sovereign-cloud/ecp/internal/provider/globalprovider"
)

// RegionHandler handles HTTP requests for region data.
type RegionHandler struct {
	provider globalprovider.RegionProvider
	logger   *slog.Logger
}

var _ regionv1.ServerInterface = (*RegionHandler)(nil) // Ensure RegionHandler implements the regionv1.ServerInterface.

// NewRegionHandler creates a new handler for region endpoints.
func NewRegionHandler(logger *slog.Logger, p globalprovider.RegionProvider) *RegionHandler {
	return &RegionHandler{provider: p, logger: logger.With("component", "RegionHandler")}
}

// ListRegions handles requests to list all available regions.
func (h *RegionHandler) ListRegions(w http.ResponseWriter, r *http.Request, params regionv1.ListRegionsParams) {
	iterator, err := h.provider.ListRegions(r.Context(), params)
	if err != nil {
		h.logger.Error("failed to list regions", "error", err)
		http.Error(w, "failed to list regions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var regions []*regionv1.Region
	if regions, err = iterator.All(r.Context()); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to retrieve all regions", slog.Any("error", err))
		http.Error(w, "failed to retrieve all regions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", string(authv1.Applicationjson))
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(regions)
	if err != nil {
		h.logger.Error("failed to encode regions", "error", err)
		http.Error(w, "failed to encode regions: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetRegion handles requests to get a specific region by name.
func (h *RegionHandler) GetRegion(w http.ResponseWriter, r *http.Request, name regionv1.ResourcePathParam) {
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

	w.Header().Set("Content-Type", string(authv1.Applicationjson))
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(reg)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to encode region", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
