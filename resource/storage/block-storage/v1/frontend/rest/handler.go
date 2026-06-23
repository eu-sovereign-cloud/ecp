package rest

import (
	"log/slog"
	"net/http"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/block-storage/v1"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/storage/storage-sku/v1"
	skurest "github.com/eu-sovereign-cloud/ecp/resource/storage/storage-sku/v1/frontend/rest"
)

// Handler is the HTTP handler for storage resources (block-storages + SKUs).
// It implements the full sdkstorage.ServerInterface.
type Handler struct {
	BlockStorageReader persistencepkg.ReaderRepo[*bsdom.BlockStorage]
	BlockStorageWriter persistencepkg.WriterRepo[*bsdom.BlockStorage]
	SKUReader          persistencepkg.ReaderRepo[*skudom.StorageSKU]
	Logger             *slog.Logger
}

var _ sdkstorage.ServerInterface = (*Handler)(nil)

// --- Images (TODO) ---

func (h *Handler) ListImages(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkstorage.ListImagesParams) {
	h.Logger.DebugContext(r.Context(), "ListImages not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkstorage.DeleteImageParams) {
	h.Logger.DebugContext(r.Context(), "DeleteImage not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) GetImage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetImage not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateOrUpdateImage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkstorage.CreateOrUpdateImageParams) {
	h.Logger.DebugContext(r.Context(), "CreateOrUpdateImage not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

// --- SKUs ---

func (h *Handler) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkstorage.ListSkusParams) {
	logger := h.Logger.With("provider", "storage", "resource", "sku")
	listParams := listSKUParamsFromAPI(params, tenant)
	frest.HandleList(w, r, logger, listParams, frest.ListerFromRepo(h.SKUReader), skurest.StorageSKUDomainToAPIIterator)
}

func (h *Handler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "storage", "resource", "sku", "name", name)
	ir := &skuIdentity{name: name, tenant: tenant}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.SKUReader, newStorageSKUWithIdentity), skurest.StorageSKUDomainToAPI)
}

// --- Block Storages ---

func (h *Handler) ListBlockStorages(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdkstorage.ListBlockStoragesParams) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage")
	frest.HandleList(w, r, logger, ListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.BlockStorageReader), BlockStorageDomainToAPIIterator)
}

func (h *Handler) DeleteBlockStorage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkstorage.DeleteBlockStorageParams) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage", "name", name)
	id := &BlockStorageIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.BlockStorageWriter, newBlockStorageWithIdentity))
}

func (h *Handler) GetBlockStorage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage", "name", name)
	ir := &BlockStorageIdentity{name: name, tenant: tenant, workspace: workspace}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.BlockStorageReader, newBlockStorageWithIdentity), BlockStorageDomainToAPIWithVerb(http.MethodGet))
}

func (h *Handler) CreateOrUpdateBlockStorage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkstorage.CreateOrUpdateBlockStorageParams) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage", "name", name)
	id := &BlockStorageIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.BlockStorage, *bsdom.BlockStorage, *sdkschema.BlockStorage]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.BlockStorageWriter),
		Updater: frest.UpdaterFromRepo(h.BlockStorageWriter),
		APIToDomain: func(sdk sdkschema.BlockStorage, p persistencepkg.IdentifiableResource) *bsdom.BlockStorage {
			return APIToBlockStorageDomain(sdk, p.(*BlockStorageIdentity), region)
		},
		DomainToAPI: BlockStorageDomainToAPIWithVerb(http.MethodPut),
	})
}

// buildListParams builds a resource.ListParams for tenant-scoped list operations.
func buildListParams(tenant, workspace string, limit *int, skipToken, selector string) resource.ListParams {
	return resource.ListParams{
		Scope: resource.Scope{
			Tenant:    tenant,
			Workspace: workspace,
		},
		Limit:     validation.GetLimit(limit),
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// listSKUParamsFromAPI converts SDK ListSkusParams to resource.ListParams.
func listSKUParamsFromAPI(params sdkstorage.ListSkusParams, tenant string) resource.ListParams {
	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}
	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}
	return buildListParams(tenant, "", params.Limit, skipToken, selector)
}

// newBlockStorageWithIdentity returns a *bsdom.BlockStorage populated with identity fields from ir.
func newBlockStorageWithIdentity(ir persistencepkg.IdentifiableResource) *bsdom.BlockStorage {
	d := &bsdom.BlockStorage{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	return d
}

// skuIdentity is a minimal IdentifiableResource for SKU get operations (tenant-scoped, no workspace).
type skuIdentity struct {
	name   string
	tenant string
}

func (s *skuIdentity) GetName() string      { return s.name }
func (s *skuIdentity) GetVersion() string   { return "" }
func (s *skuIdentity) GetTenant() string    { return s.tenant }
func (s *skuIdentity) GetWorkspace() string { return "" }

// newStorageSKUWithIdentity returns a *skudom.StorageSKU populated with identity fields from ir.
func newStorageSKUWithIdentity(ir persistencepkg.IdentifiableResource) *skudom.StorageSKU {
	d := &skudom.StorageSKU{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	return d
}
