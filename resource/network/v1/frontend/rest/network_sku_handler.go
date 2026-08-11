package rest

import (
	"net/http"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
)

// ListSkus handles GET /v1/tenants/{tenant}/skus.
func (h *Handler) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdknetwork.ListSkusParams) {
	logger := h.Logger.With("provider", "network", "resource", "sku")
	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}
	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}
	listParams := resource.ListParams{
		Scope:     resource.Scope{Tenant: tenant},
		Limit:     validation.GetLimit(params.Limit),
		SkipToken: skipToken,
		Selector:  selector,
	}
	frest.HandleList(w, r, logger, listParams, frest.ListerFromRepo(h.SKUReader), networkSKUIteratorToAPI)
}

// GetSku handles GET /v1/tenants/{tenant}/skus/{name}.
func (h *Handler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "sku")
	ir := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant}}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.SKUReader, newNetworkSKUWithIdentity), networkSKUToAPI)
}

// newNetworkSKUWithIdentity returns a *skudom.NetworkSKU populated with identity fields from ir.
func newNetworkSKUWithIdentity(ir persistencepkg.IdentifiableResource) *skudom.NetworkSKU {
	d := &skudom.NetworkSKU{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	return d
}
