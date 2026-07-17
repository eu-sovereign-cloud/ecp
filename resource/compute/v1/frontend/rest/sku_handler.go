package rest

import (
	"net/http"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku"
)

// ListSkus handles GET /v1/tenants/{tenant}/skus.
func (h *Handler) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkcompute.ListSkusParams) {
	logger := h.Logger.With("provider", "compute", "resource", "sku")
	frest.HandleList(w, r, logger, instanceSKUListParamsFromAPI(params, tenant), frest.ListerFromRepo(h.SKUReader), instanceSKUIteratorToAPI)
}

// GetSku handles GET /v1/tenants/{tenant}/skus/{name}.
func (h *Handler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "compute", "resource", "sku", "name", name)
	ir := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant}}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.SKUReader, newInstanceSKUWithIdentity), instanceSKUToAPIWithVerb(http.MethodGet))
}

// instanceSKUListParamsFromAPI converts SDK ListSkusParams to a tenant-scoped resource.ListParams.
func instanceSKUListParamsFromAPI(params sdkcompute.ListSkusParams, tenant string) resource.ListParams {
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

// newInstanceSKUWithIdentity returns a *skudom.InstanceSKU populated with identity fields from ir.
func newInstanceSKUWithIdentity(ir persistencepkg.IdentifiableResource) *skudom.InstanceSKU {
	sku := &skudom.InstanceSKU{}
	sku.Name = ir.GetName()
	sku.Tenant = ir.GetTenant()
	return sku
}
