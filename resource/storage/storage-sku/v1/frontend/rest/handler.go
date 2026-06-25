package rest

import (
	"log/slog"
	"net/http"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/storage/storage-sku/v1"
)

// SKUHandler implements the SKU methods of the storage group's
// sdkstorage.ServerInterface. It is embedded by the storage group owner
// (block-storage) Handler so the SKU methods are promoted onto it.
type SKUHandler struct {
	Reader persistencepkg.ReaderRepo[*skudom.StorageSKU]
	Logger *slog.Logger
}

// ListSkus handles GET /v1/tenants/{tenant}/skus.
func (h *SKUHandler) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkstorage.ListSkusParams) {
	logger := h.Logger.With("provider", "storage", "resource", "sku")
	frest.HandleList(w, r, logger, ListParamsFromAPI(params, tenant), frest.ListerFromRepo(h.Reader), StorageSKUIteratorToAPI)
}

// GetSku handles GET /v1/tenants/{tenant}/skus/{name}.
func (h *SKUHandler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "storage", "resource", "sku", "name", name)
	ir := &skuIdentity{name: name, tenant: tenant}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.Reader, newStorageSKUWithIdentity), StorageSKUToAPIWithVerb(http.MethodGet))
}

// ListParamsFromAPI converts SDK ListSkusParams to a tenant-scoped resource.ListParams.
func ListParamsFromAPI(params sdkstorage.ListSkusParams, tenant string) resource.ListParams {
	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}
	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}
	return resource.ListParams{
		Scope:     resource.Scope{Tenant: tenant},
		Limit:     validation.GetLimit(params.Limit),
		SkipToken: skipToken,
		Selector:  selector,
	}
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

var _ persistencepkg.IdentifiableResource = (*skuIdentity)(nil)

// newStorageSKUWithIdentity returns a *skudom.StorageSKU populated with identity fields from ir.
func newStorageSKUWithIdentity(ir persistencepkg.IdentifiableResource) *skudom.StorageSKU {
	sku := &skudom.StorageSKU{}
	sku.Name = ir.GetName()
	sku.Tenant = ir.GetTenant()
	return sku
}
