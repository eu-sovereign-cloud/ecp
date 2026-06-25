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
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/block-storage/v1"
	imgrest "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1/frontend/rest"
	skurest "github.com/eu-sovereign-cloud/ecp/resource/storage/storage-sku/v1/frontend/rest"
)

// Handler is the HTTP handler for the storage API group. It owns the group's
// sdkstorage.ServerInterface: it implements the block-storage methods directly and
// promotes the image and SKU methods from the embedded per-resource handlers.
type Handler struct {
	*imgrest.ImageHandler
	*skurest.SKUHandler
	BlockStorageReader persistencepkg.ReaderRepo[*bsdom.BlockStorage]
	BlockStorageWriter persistencepkg.WriterRepo[*bsdom.BlockStorage]
	Logger             *slog.Logger
}

var _ sdkstorage.ServerInterface = (*Handler)(nil)

// --- Block Storages ---

func (h *Handler) ListBlockStorages(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdkstorage.ListBlockStoragesParams) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage")
	frest.HandleList(w, r, logger, ListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.BlockStorageReader), BlockStorageIteratorToAPI)
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
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.BlockStorageReader, newBlockStorageWithIdentity), BlockStorageToAPIWithVerb(http.MethodGet))
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
			return BlockStorageFromAPI(sdk, p.(*BlockStorageIdentity), region)
		},
		DomainToAPI: BlockStorageToAPIWithVerb(http.MethodPut),
	})
}

// newBlockStorageWithIdentity returns a *bsdom.BlockStorage populated with identity fields from ir.
func newBlockStorageWithIdentity(ir persistencepkg.IdentifiableResource) *bsdom.BlockStorage {
	bs := &bsdom.BlockStorage{}
	bs.Name = ir.GetName()
	bs.Tenant = ir.GetTenant()
	bs.Workspace = ir.GetWorkspace()
	bs.ResourceVersion = ir.GetVersion()
	return bs
}
