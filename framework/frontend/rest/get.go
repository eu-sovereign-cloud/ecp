package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
)

// Getter defines the interface for controller Get operations.
type Getter[T any] interface {
	Do(ctx context.Context, resource persistence.IdentifiableResource) (T, error)
}

// DomainToAPI defines the function type for mapping domain objects to API objects.
type DomainToAPI[D any, Out any] func(domain D) Out

// HandleGet is a generic helper for GET endpoints that:
// 1. Calls the getter to fetch the domain object.
// 2. Handles errors via RFC 7807 response.
// 3. Maps domain to SDK.
// 4. Encodes and writes the JSON response.
func HandleGet[D any, Out any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	ir persistence.IdentifiableResource,
	getter Getter[D],
	mapper DomainToAPI[D, Out],
) {
	logger = logger.With("name", ir.GetName(), "tenant", ir.GetTenant(), "workspace", ir.GetWorkspace())

	domainObj, err := getter.Do(r.Context(), ir)
	if err != nil {
		WriteErrorResponse(w, r, logger, err)
		return
	}

	sdkObj := mapper(domainObj)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(sdkObj); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode response", slog.Any("error", err))
		WriteErrorResponse(w, r, logger, err)
		return
	}

	w.Header().Set("Content-Type", string(schema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
