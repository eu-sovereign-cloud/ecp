package globalhandler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/global/region"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/service/handler"
	regionapi "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/region"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// Region handles HTTP requests for region data.
type Region struct {
	ListRegionController *region.ListRegion
	GetRegionController  *region.GetRegion
	Logger               *slog.Logger
}

var _ regionv1.ServerInterface = (*Region)(nil)

// ListRegions handles requests to list all available regions.
func (h *Region) ListRegions(w http.ResponseWriter, r *http.Request, params regionv1.ListRegionsParams) {
	domainRegions, nextSkipToken, err := h.ListRegionController.Do(r.Context(), regionapi.ListParamsFromSDK(params))
	if err != nil {
		h.Logger.Error("failed to list regions", "error", err)
		http.Error(w, "failed to list regions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", string(schema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)

	iterator := regionapi.DomainToAPIIterator(domainRegions, nextSkipToken)

	err = json.NewEncoder(w).Encode(iterator)
	if err != nil {
		h.Logger.Error("failed to encode regions", "error", err)
		http.Error(w, "failed to encode regions: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetRegion handles requests to get a specific region by name.
func (h *Region) GetRegion(w http.ResponseWriter, r *http.Request, name schema.ResourcePathParam) {
	handler.HandleGet(w, r, h.Logger.With("resource type", "region"), &model.Metadata{
		Name: name,
	}, h.GetRegionController, regionapi.DomainToSDK)
}
