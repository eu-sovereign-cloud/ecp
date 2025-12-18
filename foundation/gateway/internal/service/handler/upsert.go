package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// Creator defines the interface for controller Create operations
type Creator[T any] interface {
	Do(ctx context.Context, resource T) (T, error)
}

// Updater defines the interface for controller Update operations
type Updater[T any] interface {
	Do(ctx context.Context, resource T) (T, error)
}

// SDKToDomain defines the function type for mapping SDK objects to domain objects
type SDKToDomain[In any, D any] func(sdk In, resourceLocator RegionalResourceLocator) D

// RegionalResourceLocator defines the interface for extracting resource location info
type RegionalResourceLocator interface {
	GetName() string
	GetTenant() string
	GetWorkspace() string
}

// ResourceLocator is a simple implementation of RegionalResourceLocator
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

// UpsertOptions contains the configuration for HandleUpsert
type UpsertOptions[In any, D any, Out any] struct {
	Locator     RegionalResourceLocator
	Creator     Creator[D]
	Updater     Updater[D]
	SDKToDomain SDKToDomain[In, D]
	DomainToSDK DomainToSDK[D, Out]
}

// HandleUpsert is a generic helper for PUT endpoints that:
// 1. Decodes the JSON request body
// 2. Maps SDK to domain
// 3. Calls the controller to create or update the resource
// 4. Handles errors appropriately
// 5. Maps domain to SDK
// 6. Encodes and writes the JSON response
func HandleUpsert[In any, D any, Out any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	options UpsertOptions[In, D, Out],
) {
	// TODO: Use workspace information from locator for resource scoping and access control
	logger = logger.With("name", options.Locator.GetName(), "tenant", options.Locator.GetTenant(), "workspace", options.Locator.GetWorkspace())

	defer func(ctx context.Context, Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.ErrorContext(ctx, "failed to close response body", "err", err)
		}
	}(r.Context(), r.Body)

	// Read and decode the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to read request body", slog.Any("error", err))
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var apiObj In
	if err := json.Unmarshal(body, &apiObj); err != nil {
		logger.ErrorContext(r.Context(), "failed to decode request body", slog.Any("error", err))
		http.Error(w, "invalid JSON in request body", http.StatusBadRequest)
		return
	}

	domainObj := options.SDKToDomain(apiObj, options.Locator)

	result, err := options.Creator.Do(r.Context(), domainObj)
	if err != nil {
		if !errors.Is(err, model.ErrAlreadyExists) {
			logger.ErrorContext(r.Context(), "failed to create resource", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// The resource already exists, so we'll attempt to update it.
		logger.InfoContext(r.Context(), "resource already exists, attempting update")
		result, err = options.Updater.Do(r.Context(), domainObj)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to update resource", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	sdkObj := options.DomainToSDK(result)
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
