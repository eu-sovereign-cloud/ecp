package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resources/common/frontend"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/domain"
	skudom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/storage-skus/v1/domain"
	skurest "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/storage-skus/v1/frontend/rest"
)

// Handler is the HTTP handler for storage resources (block-storages + SKUs).
// It implements the full sdkstorage.ServerInterface.
type Handler struct {
	BlockStorageReader persistence.ReaderRepo[*bsdom.BlockStorageDomain]
	BlockStorageWriter persistence.WriterRepo[*bsdom.BlockStorageDomain]
	SKUReader          persistence.ReaderRepo[*skudom.StorageSKUDomain]
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

	limit := params.Limit
	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}
	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	listParams := buildListParams(tenant, "", limit, skipToken, selector)

	var domains []*skudom.StorageSKUDomain
	nextSkipToken, err := h.SKUReader.List(r.Context(), listParams, &domains)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to list storage SKUs", slog.Any("error", err))
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	sdkObj := skurest.StorageSKUDomainToAPIIterator(domains, nextSkipToken)
	writeJSON(w, r, logger, sdkObj)
}

func (h *Handler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "storage", "resource", "sku", "name", name)

	domain := &skudom.StorageSKUDomain{}
	domain.Name = name
	domain.Tenant = tenant

	if err := h.SKUReader.Load(r.Context(), &domain); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	writeJSON(w, r, logger, skurest.StorageSKUDomainToAPI(domain))
}

// --- Block Storages ---

func (h *Handler) ListBlockStorages(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdkstorage.ListBlockStoragesParams) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage")
	listParams := ListParamsFromAPI(params, tenant, workspace)

	var domains []*bsdom.BlockStorageDomain
	nextSkipToken, err := h.BlockStorageReader.List(r.Context(), listParams, &domains)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to list block-storages", slog.Any("error", err))
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	writeJSON(w, r, logger, BlockStorageDomainToAPIIterator(domains, nextSkipToken))
}

func (h *Handler) DeleteBlockStorage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkstorage.DeleteBlockStorageParams) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage", "name", name)

	id := &BlockStorageIdentity{
		name:      name,
		tenant:    tenant,
		workspace: workspace,
	}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}

	domain := &bsdom.BlockStorageDomain{}
	domain.Name = id.name
	domain.Tenant = id.tenant
	domain.Workspace = id.workspace
	domain.ResourceVersion = id.resourceVersion

	if err := h.BlockStorageWriter.Delete(r.Context(), domain); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetBlockStorage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage", "name", name)

	domain := &bsdom.BlockStorageDomain{}
	domain.Name = name
	domain.Tenant = tenant
	domain.Workspace = workspace

	if err := h.BlockStorageReader.Load(r.Context(), &domain); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	toAPI := BlockStorageDomainToAPIWithVerb(http.MethodGet)
	writeJSON(w, r, logger, toAPI(domain))
}

func (h *Handler) CreateOrUpdateBlockStorage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkstorage.CreateOrUpdateBlockStorageParams) {
	logger := h.Logger.With("provider", "storage", "resource", "block-storage", "name", name)

	var resourceVersion string
	if params.IfUnmodifiedSince != nil {
		resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}
	defer func() { _ = r.Body.Close() }()

	var apiObj sdkschema.BlockStorage
	if err := json.Unmarshal(body, &apiObj); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	id := &BlockStorageIdentity{
		name:            name,
		tenant:          tenant,
		workspace:       workspace,
		resourceVersion: resourceVersion,
	}
	region := frameworkconfig.Singleton().Region()
	domainObj := APIToBlockStorageDomain(apiObj, id, region)

	var result *bsdom.BlockStorageDomain
	shouldUpdate := resourceVersion != ""
	if !shouldUpdate {
		r2, err := h.BlockStorageWriter.Create(r.Context(), domainObj)
		if err != nil {
			commonfrontend.WriteErrorResponse(w, r, logger, err)
			return
		}
		result = *r2
	} else {
		r2, err := h.BlockStorageWriter.Update(r.Context(), domainObj)
		if err != nil {
			commonfrontend.WriteErrorResponse(w, r, logger, err)
			return
		}
		result = *r2
	}

	toAPI := BlockStorageDomainToAPIWithVerb(http.MethodPut)
	writeJSON(w, r, logger, toAPI(result))
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

// writeJSON encodes v to JSON and writes it to w.
func writeJSON(w http.ResponseWriter, r *http.Request, logger *slog.Logger, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode response", slog.Any("error", err))
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}
	w.Header().Set("Content-Type", string(sdkschema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
