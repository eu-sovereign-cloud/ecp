package regionalhandler

import (
	"log/slog"
	"net/http"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/regional/storage"
	apistorage "github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/rest2domain/storage"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/service/handler"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/config"
	model "github.com/eu-sovereign-cloud/ecp/foundation/models"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional/consts"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/scope"
)

type Storage struct {
	ListSKUs           *storage.ListSKUs
	GetSKU             *storage.GetSKU
	ListStorages       *storage.ListBlockStorages
	GetStorage         *storage.GetBlockStorage
	CreateBlockStorage *storage.CreateBlockStorage
	UpdateBlockStorage *storage.UpdateBlockStorage
	DeleteStorage      *storage.DeleteBlockStorage
	ListImage          *storage.ListImages
	ImageGet           *storage.GetImage
	ImageCreate        *storage.CreateImage
	ImageUpdate        *storage.UpdateImage
	ImageDelete        *storage.DeleteImage
	Logger             *slog.Logger
}

var _ sdkstorage.ServerInterface = (*Storage)(nil) // Ensure Storage implements the sdkstorage.ServerInterface.

func (h Storage) ListImages(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	params sdkstorage.ListImagesParams,
) {
	handler.HandleList(w, r, h.Logger.With("provider", "storage").With("resource", "image"),
		apistorage.ImageListParamsFromAPI(params, tenant),
		h.ListImage,
		apistorage.ImageDomainToAPIIterator,
	)
}

func (h Storage) DeleteImage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam, params sdkstorage.DeleteImageParams,
) {
	metadata := regional.Metadata{
		CommonMetadata: model.CommonMetadata{
			Name:     name,
			Provider: consts.StorageProvider,
		},
		Scope: scope.Scope{
			Tenant: tenant,
		},
		Region: config.Singleton().Region(),
	}
	if params.IfUnmodifiedSince != nil {
		metadata.ResourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	handler.HandleDelete(w, r, h.Logger.With("provider", "storage").With("resource", "image"),
		&metadata,
		h.ImageDelete,
	)
}

func (h Storage) GetImage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam,
) {
	handler.HandleGet(w, r, h.Logger.With("provider", "storage").With("resource", "image"),
		&regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name:     name,
				Provider: consts.StorageProvider,
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
			Region: config.Singleton().Region(),
		},
		h.ImageGet,
		apistorage.ImageDomainToAPIWithVerb(http.MethodGet),
	)
}

func (h Storage) CreateOrUpdateImage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam, params sdkstorage.CreateOrUpdateImageParams,
) {
	var resourceVersion string
	if params.IfUnmodifiedSince != nil {
		resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}

	handler.HandleUpsert(w, r, h.Logger.With("provider", "storage").With("resource", "image"),
		handler.UpsertOptions[sdkschema.Image, *regional.ImageDomain, *sdkschema.Image]{
			Params: &regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					Name:            name,
					Provider:        consts.StorageProvider,
					ResourceVersion: resourceVersion,
				},
				Scope: scope.Scope{
					Tenant: tenant,
				},
				Region: config.Singleton().Region(),
			},
			Creator:     h.ImageCreate,
			Updater:     h.ImageUpdate,
			APIToDomain: apistorage.APIToImageDomain,
			DomainToAPI: apistorage.ImageDomainToAPIWithVerb(http.MethodPut),
		},
	)
}

func (h Storage) ListSkus(w http.ResponseWriter, r *http.Request,
	tenant sdkschema.TenantPathParam, params sdkstorage.ListSkusParams,
) {
	handler.HandleList(w, r, h.Logger.With("provider", "storage").With("resource", "sku"),
		apistorage.SKUListParamsFromAPI(params, tenant),
		h.ListSKUs,
		apistorage.SKUDomainToAPIIterator,
	)
}

func (h Storage) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam,
) {
	handler.HandleGet(w, r, h.Logger.With("provider", "storage").With("resource", "sku"), &regional.Metadata{
		CommonMetadata: model.CommonMetadata{
			Name:     name,
			Provider: consts.StorageProvider,
		},
		Scope: scope.Scope{
			Tenant: tenant,
		},
		Region: config.Singleton().Region(),
	}, h.GetSKU, apistorage.SKUDomainToAPI)
}

func (h Storage) ListBlockStorages(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, params sdkstorage.ListBlockStoragesParams,
) {
	handler.HandleList(w, r, h.Logger.With("provider", "storage").With("resource", "block-storage"),
		apistorage.BlockStorageListParamsFromAPI(params, tenant, workspace),
		h.ListStorages,
		apistorage.BlockStorageDomainToAPIIterator,
	)
}

func (h Storage) DeleteBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam,
	params sdkstorage.DeleteBlockStorageParams,
) {
	metadata := regional.Metadata{
		CommonMetadata: model.CommonMetadata{
			Name:     name,
			Provider: consts.StorageProvider,
		},
		Scope: scope.Scope{
			Tenant:    tenant,
			Workspace: workspace,
		},
		Region: config.Singleton().Region(),
	}
	if params.IfUnmodifiedSince != nil {
		metadata.ResourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	handler.HandleDelete(w, r, h.Logger.With("provider", "storage").With("resource", "block-storage"),
		&metadata,
		h.DeleteStorage,
	)
}

func (h Storage) GetBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam,
) {
	handler.HandleGet(w, r, h.Logger.With("provider", "storage").With("resource", "block-storage"),
		&regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name:     name,
				Provider: consts.StorageProvider,
			},
			Scope: scope.Scope{
				Tenant:    tenant,
				Workspace: workspace,
			},
			Region: config.Singleton().Region(),
		},
		h.GetStorage,
		apistorage.BlockStorageDomainToAPIWithVerb(http.MethodGet),
	)
}

func (h Storage) CreateOrUpdateBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam,
	params sdkstorage.CreateOrUpdateBlockStorageParams,
) {
	var resourceVersion string
	if params.IfUnmodifiedSince != nil {
		resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}

	handler.HandleUpsert(w, r, h.Logger.With("provider", "storage").With("resource", "block-storage"),
		handler.UpsertOptions[sdkschema.BlockStorage, *regional.BlockStorageDomain, *sdkschema.BlockStorage]{
			Params: &regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					Name:            name,
					Provider:        consts.StorageProvider,
					ResourceVersion: resourceVersion,
				},
				Scope: scope.Scope{
					Tenant:    tenant,
					Workspace: workspace,
				},
				Region: config.Singleton().Region(),
			},
			Creator:     h.CreateBlockStorage,
			Updater:     h.UpdateBlockStorage,
			APIToDomain: apistorage.APIToBlockStorageDomain,
			DomainToAPI: apistorage.BlockStorageDomainToAPIWithVerb(http.MethodPut),
		},
	)
}
