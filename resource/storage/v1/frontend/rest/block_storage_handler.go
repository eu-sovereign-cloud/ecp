package rest

import (
	"net/http"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

// ListBlockStorages handles GET /v1/tenants/{tenant}/workspaces/{workspace}/block-storages.
func (h *Handler) ListBlockStorages(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdkstorage.ListBlockStoragesParams) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage")
	frest.HandleList(w, r, logger, blockStorageListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.BlockStorageReader), BlockStorageIteratorToAPI)
}

// DeleteBlockStorage handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/block-storages/{name}.
func (h *Handler) DeleteBlockStorage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkstorage.DeleteBlockStorageParams) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage", "name", name)
	id := &BlockStorageIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.BlockStorageWriter, newBlockStorageWithIdentity))
}

// GetBlockStorage handles GET /v1/tenants/{tenant}/workspaces/{workspace}/block-storages/{name}.
func (h *Handler) GetBlockStorage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage", "name", name)
	ir := &BlockStorageIdentity{name: name, tenant: tenant, workspace: workspace}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.BlockStorageReader, newBlockStorageWithIdentity), BlockStorageToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateBlockStorage handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/block-storages/{name}.
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
