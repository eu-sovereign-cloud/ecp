package rest

import (
	"net/http"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

// ListRouteTables handles GET /v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/route-tables.
func (h *Handler) ListRouteTables(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, params sdknetwork.ListRouteTablesParams) {
	logger := h.Logger.With("provider", "network", "resource", "route-table")
	frest.HandleList(w, r, logger, routeTableListParamsFromAPI(params, tenant, workspace, network), frest.ListerFromRepo(h.RouteTableReader), routeTableIteratorToAPI)
}

// DeleteRouteTable handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/route-tables/{name}.
func (h *Handler) DeleteRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteRouteTableParams) {
	logger := h.Logger.With("provider", "network", "resource", "route-table", "name", name)
	id := &RouteTableIdentity{name: name, tenant: tenant, workspace: workspace, network: network}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.RouteTableWriter, newRouteTableWithIdentity))
}

// GetRouteTable handles GET /v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/route-tables/{name}.
func (h *Handler) GetRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "route-table", "name", name)
	ir := &RouteTableIdentity{name: name, tenant: tenant, workspace: workspace, network: network}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.RouteTableReader, newRouteTableWithIdentity), routeTableToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateRouteTable handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/route-tables/{name}.
func (h *Handler) CreateOrUpdateRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateRouteTableParams) {
	logger := h.Logger.With("provider", "network", "resource", "route-table", "name", name)
	id := &RouteTableIdentity{name: name, tenant: tenant, workspace: workspace, network: network}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.RouteTable, *routetabledom.RouteTable, *sdkschema.RouteTable]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.RouteTableWriter),
		Updater: frest.UpdaterFromRepo(h.RouteTableWriter),
		APIToDomain: func(sdk sdkschema.RouteTable, p persistencepkg.IdentifiableResource) *routetabledom.RouteTable {
			return routeTableFromAPI(sdk, p.(*RouteTableIdentity), region)
		},
		DomainToAPI: routeTableToAPIWithVerb(http.MethodPut),
	})
}
