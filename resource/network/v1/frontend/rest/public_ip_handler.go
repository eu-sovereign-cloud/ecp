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
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

// ListPublicIps handles GET /v1/tenants/{tenant}/workspaces/{workspace}/public-ips.
func (h *Handler) ListPublicIps(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListPublicIpsParams) {
	logger := h.Logger.With("provider", "network", "resource", "public-ip")
	frest.HandleList(w, r, logger, publicIpListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.PublicIpReader), publicIpIteratorToAPI)
}

// DeletePublicIp handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/public-ips/{name}.
func (h *Handler) DeletePublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeletePublicIpParams) {
	logger := h.Logger.With("provider", "network", "resource", "public-ip", "name", name)
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.PublicIpWriter, newPublicIpWithIdentity))
}

// GetPublicIp handles GET /v1/tenants/{tenant}/workspaces/{workspace}/public-ips/{name}.
func (h *Handler) GetPublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "public-ip", "name", name)
	ir := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.PublicIpReader, newPublicIpWithIdentity), publicIpToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdatePublicIp handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/public-ips/{name}.
func (h *Handler) CreateOrUpdatePublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdatePublicIpParams) {
	logger := h.Logger.With("provider", "network", "resource", "public-ip", "name", name)
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.PublicIp, *publicipdom.PublicIp, *sdkschema.PublicIp]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.PublicIpWriter),
		Updater: frest.UpdaterFromRepo(h.PublicIpWriter),
		APIToDomain: func(sdk sdkschema.PublicIp, p persistencepkg.IdentifiableResource) *publicipdom.PublicIp {
			return publicIpFromAPI(sdk, p.(*resource.Identity), region)
		},
		DomainToAPI: publicIpToAPIWithVerb(http.MethodPut),
	})
}
