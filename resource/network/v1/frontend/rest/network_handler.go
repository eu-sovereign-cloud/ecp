package rest

import (
	"net/http"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

// ListNetworks handles GET /v1/tenants/{tenant}/workspaces/{workspace}/networks.
func (h *Handler) ListNetworks(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListNetworksParams) {
	logger := h.Logger.With("provider", "network", "resource", "network")
	frest.HandleList(w, r, logger, networkListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.NetworkReader), networkIteratorToAPI)
}

// DeleteNetwork handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/networks/{name}.
func (h *Handler) DeleteNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteNetworkParams) {
	logger := h.Logger.With("provider", "network", "resource", "network")
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.NetworkWriter, newNetworkWithIdentity))
}

// GetNetwork handles GET /v1/tenants/{tenant}/workspaces/{workspace}/networks/{name}.
func (h *Handler) GetNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "network")
	ir := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.NetworkReader, newNetworkWithIdentity), networkToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateNetwork handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/networks/{name}.
func (h *Handler) CreateOrUpdateNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateNetworkParams) {
	logger := h.Logger.With("provider", "network", "resource", "network")
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Network, *netdom.Network, *sdkschema.Network]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.NetworkWriter),
		Updater: frest.UpdaterFromRepo(h.NetworkWriter),
		APIToDomain: func(sdk sdkschema.Network, p persistencepkg.IdentifiableResource) (*netdom.Network, error) {
			return networkFromAPI(sdk, p.(*resource.Identity), region)
		},
		DomainToAPI: networkToAPIWithVerb(http.MethodPut),
	})
}

// newNetworkWithIdentity returns a *netdom.Network populated with identity fields from ir.
func newNetworkWithIdentity(ir persistencepkg.IdentifiableResource) *netdom.Network {
	d := &netdom.Network{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	return d
}
