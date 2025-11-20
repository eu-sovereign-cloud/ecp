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
	ListSKUs *storage.ListSKUs
	GetSKU   *storage.GetSKU
	Logger   *slog.Logger
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
	modelParams := apistorage.ListParamsFromAPI(params)
	modelParams.Namespace = tenant
	handler.HandleList(w, r, h.Logger.With("provider", "storage").With("resource", "sku"),
		modelParams,
		h.ListSKUs,
		apistorage.SKUDomainsToAPIIterator,
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
	// TODO implement me
}
