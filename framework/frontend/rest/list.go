package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

// Lister defines the interface for controller List operations.
type Lister[T any] interface {
	Do(ctx context.Context, params resource.ListParams) ([]T, *string, error)
}

// DomainToAPIList defines the function type for mapping a list of domain objects to an API object.
type DomainToAPIList[D any, Out any] func(domain []D, nextSkipToken *string) Out

// HandleList is a generic helper for LIST endpoints that:
// 1. Calls the lister to fetch the list of domain objects.
// 2. Handles errors via RFC 7807 response.
// 3. Maps the domain list to an SDK object.
// 4. Encodes and writes the JSON response.
func HandleList[D any, Out any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	params resource.ListParams,
	lister Lister[D],
	mapper DomainToAPIList[D, Out],
) {
	domainObjs, nextSkipToken, err := lister.Do(r.Context(), params)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to list resources", slog.Any("error", err))
		WriteErrorResponse(w, r, logger, err)
		return
	}

	sdkObj := mapper(domainObjs, nextSkipToken)
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
