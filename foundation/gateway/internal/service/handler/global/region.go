package globalhandler

import (
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
	handler.HandleList(w, r, h.Logger.With("provider", "region").With("resource", "region"),
		regionapi.ListParamsFromSDK(params),
		h.ListRegionController,
		regionapi.DomainToAPIIterator,
	)
}

// GetRegion handles requests to get a specific region by name.
func (h *Region) GetRegion(w http.ResponseWriter, r *http.Request, name schema.ResourcePathParam) {
	handler.HandleGet(w, r, h.Logger.With("provider", "region").With("resource", "region"), &model.Metadata{
		Name: name,
	}, h.GetRegionController, regionapi.DomainToSDK)
}
