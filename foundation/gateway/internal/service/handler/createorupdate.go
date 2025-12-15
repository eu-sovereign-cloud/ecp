package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"k8s.io/apimachinery/pkg/api/errors"
)

// Creator defines the interface for controller Create operations
type Creator[T any] interface {
	Do(ctx context.Context, resource T) (T, error)
}

// APIToDomain defines the interface for mapping API objects to domain objects
type APIToDomain[In any, D any] func(api In, tenant, name string) D

type RegionalResourceLocator interface {
	GetName() string
	GetTenant() string
	GetWorkspace() string
}

type ResourceLocator struct {
	Name      string
	Tenant    string
	Workspace string
}

func (r ResourceLocator) GetName() string {
	return r.Name
}

func (r ResourceLocator) GetTenant() string {
	return r.Tenant
}

func (r ResourceLocator) GetWorkspace() string {
	return r.Workspace
}

// HandleCreateOrUpdate is a generic helper for PUT endpoints that:
// 1. Decodes the JSON request body
// 2. Maps API to domain
// 3. Calls the controller to create or update the resource
// 4. Handles errors appropriately
// 5. Maps domain to SDK
// 6. Encodes and writes the JSON response
func HandleCreateOrUpdate[In any, D any, Out any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	path RegionalResourceLocator,
	creator Creator[D],
	apiToDomain APIToDomain[In, D],
	domainToAPI DomainToAPI[D, Out],
) {
	name := path.GetName()
	tenant := path.GetTenant()
	logger = logger.With("name", name, "tenant", tenant)

	// Read and decode the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to read request body", slog.Any("error", err))
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var apiObj In
	if err := json.Unmarshal(body, &apiObj); err != nil {
		logger.ErrorContext(r.Context(), "failed to decode request body", slog.Any("error", err))
		http.Error(w, "invalid JSON in request body", http.StatusBadRequest)
		return
	}

	// Convert API to domain
	domainObj := apiToDomain(apiObj, tenant, name)

	// Create or update the resource
	result, err := creator.Do(r.Context(), domainObj)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Resource already exists, might need update logic
			logger.InfoContext(r.Context(), "resource already exists")
			http.Error(w, fmt.Sprintf("resource %s already exists", name), http.StatusConflict)
			return
		}

		// For all other errors, log and return 500
		logger.ErrorContext(r.Context(), "failed to create or update", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Convert domain back to API
	sdkObj := domainToAPI(result)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(sdkObj); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode response", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", string(schema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
