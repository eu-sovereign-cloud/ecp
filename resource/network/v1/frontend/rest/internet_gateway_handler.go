package rest

import (
	"net/http"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// ListInternetGateways handles GET /v1/tenants/{tenant}/workspaces/{workspace}/internet-gateways.
func (h *Handler) ListInternetGateways(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListInternetGatewaysParams) {
	logger := h.Logger.With("provider", "network", "resource", "internet-gateway")
	frest.HandleList(w, r, logger, internetGatewayListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.InternetGatewayReader), internetGatewayIteratorToAPI)
}

// DeleteInternetGateway handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/internet-gateways/{name}.
func (h *Handler) DeleteInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteInternetGatewayParams) {
	logger := h.Logger.With("provider", "network", "resource", "internet-gateway", "name", name)
	id := &InternetGatewayIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.InternetGatewayWriter, newInternetGatewayWithIdentity))
}

// GetInternetGateway handles GET /v1/tenants/{tenant}/workspaces/{workspace}/internet-gateways/{name}.
func (h *Handler) GetInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "internet-gateway", "name", name)
	ir := &InternetGatewayIdentity{name: name, tenant: tenant, workspace: workspace}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.InternetGatewayReader, newInternetGatewayWithIdentity), internetGatewayToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateInternetGateway handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/internet-gateways/{name}.
func (h *Handler) CreateOrUpdateInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateInternetGatewayParams) {
	logger := h.Logger.With("provider", "network", "resource", "internet-gateway", "name", name)
	id := &InternetGatewayIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.InternetGateway, *internetgatewaydom.InternetGateway, *sdkschema.InternetGateway]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.InternetGatewayWriter),
		Updater: frest.UpdaterFromRepo(h.InternetGatewayWriter),
		APIToDomain: func(sdk sdkschema.InternetGateway, p persistencepkg.IdentifiableResource) *internetgatewaydom.InternetGateway {
			return internetGatewayFromAPI(sdk, p.(*InternetGatewayIdentity), region)
		},
		DomainToAPI: internetGatewayToAPIWithVerb(http.MethodPut),
	})
}
