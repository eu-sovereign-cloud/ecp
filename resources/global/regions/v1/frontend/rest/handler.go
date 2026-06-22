package rest

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"

	regionv1sdk "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resources/common/frontend"
	rdom "github.com/eu-sovereign-cloud/ecp/resources/global/regions/v1"
)

// Handler is the HTTP handler for region resources.
type Handler struct {
	Repo   persistence.ReaderRepo[*rdom.Region]
	Logger *slog.Logger
}

var _ regionv1sdk.ServerInterface = (*Handler)(nil)

// ListRegions handles GET /v1/regions.
func (h *Handler) ListRegions(w http.ResponseWriter, r *http.Request, params regionv1sdk.ListRegionsParams) {
	logger := h.Logger.With("resource", "region")
	listParams := ListParamsFromAPI(params)

	var domains []*rdom.Region
	nextSkipToken, err := h.Repo.List(r.Context(), listParams, &domains)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to list regions", slog.Any("error", err))
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	sdkObj := DomainToAPIIterator(domains, nextSkipToken)
	writeJSON(w, r, logger, sdkObj)
}

// GetRegion handles GET /v1/regions/{name}.
func (h *Handler) GetRegion(w http.ResponseWriter, r *http.Request, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("resource", "region", "name", name)

	domain := &rdom.Region{}
	domain.Name = name

	if err := h.Repo.Load(r.Context(), &domain); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	sdkObj := DomainToAPI(domain)
	writeJSON(w, r, logger, sdkObj)
}

// writeJSON encodes v to JSON and writes it to w with Content-Type: application/json.
func writeJSON(w http.ResponseWriter, r *http.Request, logger *slog.Logger, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode response", slog.Any("error", err))
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}
	w.Header().Set("Content-Type", string(sdkschema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
