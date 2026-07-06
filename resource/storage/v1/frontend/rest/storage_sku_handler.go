package rest

import (
	"net/http"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
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
	ir := &commondomain.RegionalMetadata{
		CommonMetadata: commondomain.CommonMetadata{Name: name},
		Scope:          resource.Scope{Tenant: tenant},
	}
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

// newStorageSKUWithIdentity returns a *skudom.StorageSKU populated with identity fields from ir.
func newStorageSKUWithIdentity(ir persistencepkg.IdentifiableResource) *skudom.StorageSKU {
	sku := &skudom.StorageSKU{}
	sku.Name = ir.GetName()
	sku.Tenant = ir.GetTenant()
	return sku
}
