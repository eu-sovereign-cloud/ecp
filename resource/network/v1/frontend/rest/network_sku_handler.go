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
	frest.HandleList(w, r, logger, listParams, frest.ListerFromRepo(h.SKUReader), NetworkSKUIteratorToAPI)
}

// GetSku handles GET /v1/tenants/{tenant}/skus/{name}.
func (h *Handler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "sku", "name", name)
	ir := &networkSKUIdentity{name: name, tenant: tenant}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.SKUReader, newNetworkSKUWithIdentity), NetworkSKUToAPI)
}

// networkSKUIdentity is a minimal IdentifiableResource for network-SKU get operations.
type networkSKUIdentity struct {
	name   string
	tenant string
}

func (s *networkSKUIdentity) GetName() string      { return s.name }
func (s *networkSKUIdentity) GetVersion() string   { return "" }
func (s *networkSKUIdentity) GetTenant() string    { return s.tenant }
func (s *networkSKUIdentity) GetWorkspace() string { return "" }

var _ persistencepkg.IdentifiableResource = (*networkSKUIdentity)(nil)

// newNetworkSKUWithIdentity returns a *skudom.NetworkSKU populated with identity fields from ir.
func newNetworkSKUWithIdentity(ir persistencepkg.IdentifiableResource) *skudom.NetworkSKU {
	d := &skudom.NetworkSKU{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	return d
}
