package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
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
		if errors.IsNotFound(err) {
			logger.InfoContext(r.Context(), "not found")
			http.Error(w, fmt.Sprintf("%s not found", ir.GetName()), http.StatusNotFound)
			return
		}

		// For all other errors (e.g., connection issues, CRD not registered),
		// log the error and return a 500 Internal Server Error.
		logger.ErrorContext(r.Context(), "failed to get", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sdkObj := mapper(domainObj)
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
