package rest

import (
	"net/http"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

// ListNics handles GET /v1/tenants/{tenant}/workspaces/{workspace}/nics.
func (h *Handler) ListNics(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListNicsParams) {
	logger := h.Logger.With("provider", "network", "resource", "nic")
	frest.HandleList(w, r, logger, nicListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.NicReader), NicIteratorToAPI)
}

// DeleteNic handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/nics/{name}.
func (h *Handler) DeleteNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteNicParams) {
	logger := h.Logger.With("provider", "network", "resource", "nic", "name", name)
	id := &NicIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.NicWriter, newNicWithIdentity))
}

// GetNic handles GET /v1/tenants/{tenant}/workspaces/{workspace}/nics/{name}.
func (h *Handler) GetNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "nic", "name", name)
	ir := &NicIdentity{name: name, tenant: tenant, workspace: workspace}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.NicReader, newNicWithIdentity), NicToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateNic handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/nics/{name}.
func (h *Handler) CreateOrUpdateNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateNicParams) {
	logger := h.Logger.With("provider", "network", "resource", "nic", "name", name)
	id := &NicIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Nic, *nicdom.Nic, *sdkschema.Nic]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.NicWriter),
		Updater: frest.UpdaterFromRepo(h.NicWriter),
		APIToDomain: func(sdk sdkschema.Nic, p persistencepkg.IdentifiableResource) *nicdom.Nic {
			return NicFromAPI(sdk, p.(*NicIdentity), region)
		},
		DomainToAPI: NicToAPIWithVerb(http.MethodPut),
	})
}
