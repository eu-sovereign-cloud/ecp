package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// Lister defines the interface for controller List operations.
type Lister[T any] interface {
	Do(ctx context.Context, params model.ListParams) ([]T, *string, error)
}

// DomainToSDKList defines the interface for mapping a list of domain objects to an SDK object.
type DomainToSDKList[D any, Out any] func(domain []D, nextSkipToken *string) Out

// HandleList is a generic helper for LIST endpoints that:
// 1. Calls the controller to fetch the list of domain objects.
// 2. Handles errors with 500.
// 3. Maps the domain list to an SDK object.
// 4. Encodes and writes the JSON response.
func HandleList[D any, Out any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	params model.ListParams,
	lister Lister[D],
	mapper DomainToSDKList[D, Out],
) {
	domainObjs, nextSkipToken, err := lister.Do(r.Context(), params)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to list", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sdkObj := mapper(domainObjs, nextSkipToken)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(sdkObj); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", string(schema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
