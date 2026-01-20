package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	apierr "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/errors"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// Getter defines the interface for controller Get operations
type Getter[T any] interface {
	Do(ctx context.Context, resource port.IdentifiableResource) (T, error)
}

// DomainToSDK defines the interface for mapping domain objects to SDK objects
type DomainToSDK[D any, Out any] func(domain D) Out

// HandleGet is a generic helper for GET endpoints that:
// 1. Calls the controller to fetch the domain object
// 2. Handles not found errors with 404
// 3. Handles other errors with 500
// 4. Maps domain to SDK
// 5. Encodes and writes the JSON response
func HandleGet[D any, Out any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	ir port.IdentifiableResource,
	getter Getter[D],
	mapper DomainToSDK[D, Out],
) {
	logger = logger.With("name", ir.GetName(), "tenant", ir.GetTenant(), "workspace", ir.GetWorkspace())

	domainObj, err := getter.Do(r.Context(), ir)
	if err != nil {
		apierr.WriteErrorResponse(w, r, logger, err)
		return
	}

	sdkObj := mapper(domainObj)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(sdkObj); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode response", slog.Any("error", err))
		apierr.WriteErrorResponse(w, r, logger, err)
		return
	}

	w.Header().Set("Content-Type", string(schema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
