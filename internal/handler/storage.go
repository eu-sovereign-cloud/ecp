package handler

import (
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"

    authv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
    sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
    "k8s.io/apimachinery/pkg/api/errors"

    "github.com/eu-sovereign-cloud/ecp/internal/provider/regionalprovider"
)

type StorageHandler struct{
	provider regionalprovider.StorageProvider
	logger   *slog.Logger
}

var _ sdkstorage.ServerInterface = (*StorageHandler)(nil) // Ensure StorageHandler implements the sdkstorage.ServerInterface.

func NewStorageHandler(logger *slog.Logger, p regionalprovider.StorageProvider) *StorageHandler {
	return &StorageHandler{provider: p, logger: logger.With("component", "StorageHandler")}
}

func (h StorageHandler) ListImages(
	w http.ResponseWriter, r *http.Request, tenant sdkstorage.TenantPathParam,
	params sdkstorage.ListImagesParams,
) {
	// TODO implement me
	panic("implement me")
}

func (h StorageHandler) DeleteImage(
	w http.ResponseWriter, r *http.Request, tenant sdkstorage.TenantPathParam,
	name sdkstorage.ResourcePathParam, params sdkstorage.DeleteImageParams,
) {
	// TODO implement me
	panic("implement me")
}

func (h StorageHandler) GetImage(
	w http.ResponseWriter, r *http.Request, tenant sdkstorage.TenantPathParam,
	name sdkstorage.ResourcePathParam,
) {
	// TODO implement me
	panic("implement me")
}

func (h StorageHandler) CreateOrUpdateImage(
	w http.ResponseWriter, r *http.Request, tenant sdkstorage.TenantPathParam,
	name sdkstorage.ResourcePathParam, params sdkstorage.CreateOrUpdateImageParams,
) {
	// TODO implement me
	panic("implement me")
}

func (h StorageHandler) ListSkus(
	w http.ResponseWriter, r *http.Request, tenant sdkstorage.TenantPathParam,
	params sdkstorage.ListSkusParams,
) {
    iterator, err := h.provider.ListSKUs(r.Context(), tenant, params)
    if err != nil {
        h.logger.Error("failed to list storage skus", "error", err)
        http.Error(w, "failed to list storage skus: "+err.Error(), http.StatusInternalServerError)
        return
    }

    var storageSKUs []*sdkstorage.StorageSku
    if storageSKUs, err = iterator.All(r.Context()); err != nil {
        h.logger.ErrorContext(r.Context(), "failed to retrieve all storage skus", slog.Any("error", err))
        http.Error(w, "failed to retrieve all storage skus: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", string(authv1.Applicationjson))
    w.WriteHeader(http.StatusOK)
    err = json.NewEncoder(w).Encode(storageSKUs)
    if err != nil {
        h.logger.Error("failed to encode storage skus", "error", err)
        http.Error(w, "failed to encode storage skus: "+err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h StorageHandler) GetSku(
	w http.ResponseWriter, r *http.Request, tenant sdkstorage.TenantPathParam,
	name sdkstorage.ResourcePathParam,
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
        h.logger.ErrorContext(r.Context(), "failed to get storage sku", slog.String("sku", name), slog.Any("error", err))
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", string(authv1.Applicationjson))
    w.WriteHeader(http.StatusOK)
    err = json.NewEncoder(w).Encode(sku)
    if err != nil {
        h.logger.ErrorContext(r.Context(), "failed to encode storage sku", slog.Any("error", err))
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        return
    }
}

func (h StorageHandler) ListBlockStorages(
	w http.ResponseWriter, r *http.Request, tenant sdkstorage.TenantPathParam,
	workspace sdkstorage.WorkspacePathParam, params sdkstorage.ListBlockStoragesParams,
) {
	// TODO implement me
	panic("implement me")
}

func (h StorageHandler) DeleteBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkstorage.TenantPathParam,
	workspace sdkstorage.WorkspacePathParam, name sdkstorage.ResourcePathParam,
	params sdkstorage.DeleteBlockStorageParams,
) {
	// TODO implement me
	panic("implement me")
}

func (h StorageHandler) GetBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkstorage.TenantPathParam,
	workspace sdkstorage.WorkspacePathParam, name sdkstorage.ResourcePathParam,
) {
	// TODO implement me
	panic("implement me")
}

func (h StorageHandler) CreateOrUpdateBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkstorage.TenantPathParam,
	workspace sdkstorage.WorkspacePathParam, name sdkstorage.ResourcePathParam,
	params sdkstorage.CreateOrUpdateBlockStorageParams,
) {
	var blockStorage sdkstorage.BlockStorage
    if err := json.NewDecoder(r.Body).Decode(&blockStorage); err != nil {
        h.logger.ErrorContext(r.Context(), "failed to decode request body", slog.Any("error", err))
        http.Error(w, "failed to decode request body: "+err.Error(), http.StatusBadRequest)
        return
    }

    createdBlockStorage, wasUpdated, err := h.provider.CreateOrUpdateBlockStorage(r.Context(), tenant, workspace, name, params, blockStorage)
    if err != nil {
        if errors.IsBadRequest(err) {
            h.logger.InfoContext(r.Context(), "bad request for creating or updating block storage", slog.Any("error", err))
            http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
            return
        }

        h.logger.ErrorContext(r.Context(), "failed to create or update block storage", slog.Any("error", err))
        http.Error(w, "failed to create or update block storage: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", string(authv1.Applicationjson))
    if wasUpdated {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusCreated)
    }
    if err := json.NewEncoder(w).Encode(createdBlockStorage); err != nil {
        h.logger.ErrorContext(r.Context(), "failed to encode response body", slog.Any("error", err))
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        return
    }
}
