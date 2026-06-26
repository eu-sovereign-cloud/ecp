package rest

import (
	"net/http"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku"
)

// ListSkus handles GET /v1/tenants/{tenant}/skus.
func (h *Handler) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkstorage.ListSkusParams) {
	logger := h.Logger.With("provider", "storage", "resource", "sku")
	frest.HandleList(w, r, logger, storageSKUListParamsFromAPI(params, tenant), frest.ListerFromRepo(h.SKUReader), StorageSKUIteratorToAPI)
}

// GetSku handles GET /v1/tenants/{tenant}/skus/{name}.
func (h *Handler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "storage", "resource", "sku", "name", name)
	ir := &skuIdentity{name: name, tenant: tenant}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.SKUReader, newStorageSKUWithIdentity), StorageSKUToAPIWithVerb(http.MethodGet))
}

// storageSKUListParamsFromAPI converts SDK ListSkusParams to a tenant-scoped resource.ListParams.
func storageSKUListParamsFromAPI(params sdkstorage.ListSkusParams, tenant string) resource.ListParams {
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
