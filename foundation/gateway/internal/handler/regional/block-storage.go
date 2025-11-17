package regionalhandler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/port"
)

type Storage struct {
	provider port.StorageProvider
	logger   *slog.Logger
}

var _ sdkstorage.ServerInterface = (*Storage)(nil) // Ensure Storage implements the sdkstorage.ServerInterface.

func NewStorage(logger *slog.Logger, p port.StorageProvider) *Storage {
	return &Storage{provider: p, logger: logger.With("component", "Storage")}
}

func (h Storage) ListImages(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	params sdkstorage.ListImagesParams,
) {
	// TODO implement me
	panic("implement me")
}

func (h Storage) DeleteImage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam, params sdkstorage.DeleteImageParams,
) {
	// TODO implement me
	panic("implement me")
}

func (h Storage) GetImage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam,
) {
	// TODO implement me
	panic("implement me")
}

func (h Storage) CreateOrUpdateImage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam, params sdkstorage.CreateOrUpdateImageParams,
) {
	// TODO implement me
	panic("implement me")
}

func (h Storage) ListSkus(w http.ResponseWriter, r *http.Request,
	tenant sdkschema.TenantPathParam, params sdkstorage.ListSkusParams,
) {
	iterator, err := h.provider.ListSKUs(r.Context(), tenant, params)
	if err != nil {
		h.logger.Error("failed to list storage skus", "error", err)
		http.Error(w, "failed to list storage skus: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", string(sdkschema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(iterator)
	if err != nil {
		h.logger.Error("failed to encode storage skus", "error", err)
		http.Error(w, "failed to encode storage skus: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h Storage) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam,
) {
	sku, err := h.provider.GetSKU(r.Context(), tenant, name)
	if err != nil {
		if errors.IsNotFound(err) {
			h.logger.InfoContext(r.Context(), "storage sku not found", slog.String("sku", name))
			http.Error(w, fmt.Sprintf("storage sku (%s) not found", name), http.StatusNotFound)
			return
		}

		// For all other errors (e.g., connection issues, CRD not registered),
		// log the error and return a 500 Internal Server Error.
		h.logger.ErrorContext(
			r.Context(), "failed to get storage sku", slog.String("sku", name), slog.Any("error", err),
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", string(sdkschema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(sku)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to encode storage sku", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h Storage) ListBlockStorages(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, params sdkstorage.ListBlockStoragesParams,
) {
	// TODO implement me
	panic("implement me")
}

func (h Storage) DeleteBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam,
	params sdkstorage.DeleteBlockStorageParams,
) {
	// TODO implement me
	panic("implement me")
}

func (h Storage) GetBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam,
) {
	// TODO implement me
	panic("implement me")
}

func (h Storage) CreateOrUpdateBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam,
	params sdkstorage.CreateOrUpdateBlockStorageParams,
) {
	// TODO implement me
	panic("implement me")
}
