package rest

import (
	"log/slog"
	"net/http"

	regionv1sdk "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	rdom "github.com/eu-sovereign-cloud/ecp/resource/region/v1"
)

// Handler is the HTTP handler for region resources.
type Handler struct {
	Repo   persistencepkg.ReaderRepo[*rdom.Region]
	Logger *slog.Logger
}

var _ regionv1sdk.ServerInterface = (*Handler)(nil)

// ListRegions handles GET /v1/regions.
func (h *Handler) ListRegions(w http.ResponseWriter, r *http.Request, params regionv1sdk.ListRegionsParams) {
	logger := h.Logger.With("resource", "region")
	frest.HandleList(w, r, logger, ListParamsFromAPI(params), frest.ListerFromRepo(h.Repo), DomainToAPIIterator)
}

// GetRegion handles GET /v1/regions/{name}.
func (h *Handler) GetRegion(w http.ResponseWriter, r *http.Request, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("resource", "region", "name", name)
	ir := &regionIdentity{name: name}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.Repo, newRegionWithIdentity), DomainToAPI)
}

// regionIdentity is a minimal IdentifiableResource for region get operations (global, no tenant/workspace).
type regionIdentity struct {
	name string
}

func (r *regionIdentity) GetName() string      { return r.name }
func (r *regionIdentity) GetVersion() string   { return "" }
func (r *regionIdentity) GetTenant() string    { return "" }
func (r *regionIdentity) GetWorkspace() string { return "" }

// newRegionWithIdentity returns a *rdom.Region populated with identity fields from ir.
func newRegionWithIdentity(ir persistencepkg.IdentifiableResource) *rdom.Region {
	d := &rdom.Region{}
	d.Name = ir.GetName()
	return d
}
