package regionalhandler

import (
	"log/slog"
	"net/http"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/regional/storage"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/service/handler"
	apistorage "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/storage"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

type Storage struct {
	ListSKUs       *storage.ListSKUs
	GetSKU         *storage.GetSKU
	CreateInstance *storage.CreateOrUpdateInstance
	Logger         *slog.Logger
}

var _ sdkstorage.ServerInterface = (*Storage)(nil) // Ensure Storage implements the sdkstorage.ServerInterface.

func (h Storage) ListImages(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	params sdkstorage.ListImagesParams,
) {
	// TODO implement me
}

func (h Storage) DeleteImage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam, params sdkstorage.DeleteImageParams,
) {
	// TODO implement me
}

func (h Storage) GetImage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam,
) {
	// TODO implement me
}

func (h Storage) CreateOrUpdateImage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam, params sdkstorage.CreateOrUpdateImageParams,
) {
	// TODO implement me
}

func (h Storage) ListSkus(w http.ResponseWriter, r *http.Request,
	tenant sdkschema.TenantPathParam, params sdkstorage.ListSkusParams,
) {
	handler.HandleList(w, r, h.Logger.With("provider", "storage").With("resource", "sku"),
		apistorage.ListParamsFromAPI(params, tenant),
		h.ListSKUs,
		apistorage.SKUDomainToAPIIterator,
	)
}

func (h Storage) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	name sdkschema.ResourcePathParam,
) {
	handler.HandleGet(w, r, h.Logger.With("provider", "storage").With("resource", "sku"), &model.Metadata{
		Name:      name,
		Namespace: tenant,
	}, h.GetSKU, apistorage.SkuToApi)
}

func (h Storage) ListBlockStorages(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, params sdkstorage.ListBlockStoragesParams,
) {
	// TODO implement me
}

func (h Storage) DeleteBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam,
	params sdkstorage.DeleteBlockStorageParams,
) {
	// TODO implement me
}

func (h Storage) GetBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam,
) {
	// TODO implement me
}

func (h Storage) CreateOrUpdateBlockStorage(
	w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam,
	workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam,
	params sdkstorage.CreateOrUpdateBlockStorageParams,
) {
	handler.HandleCreateOrUpdate(w, r, h.Logger.With("provider", "storage").With("resource", "instance").With("name", name).With("tenant", tenant),
		handler.ResourceLocator{
			Name:      name,
			Tenant:    tenant,
			Workspace: workspace,
		},
		h.CreateInstance,
		apistorage.BlockStorageFromAPI,
		apistorage.BlockStorageToAPI,
	)
}
