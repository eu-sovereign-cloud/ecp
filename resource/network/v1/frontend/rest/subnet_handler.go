package rest

import (
	"net/http"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

// ListSubnets handles GET /v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/subnets.
func (h *Handler) ListSubnets(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, params sdknetwork.ListSubnetsParams) {
	logger := h.Logger.With("provider", "network", "resource", "subnet")
	frest.HandleList(w, r, logger, subnetListParamsFromAPI(params, tenant, workspace, network), frest.ListerFromRepo(h.SubnetReader), subnetIteratorToAPI)
}

// DeleteSubnet handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/subnets/{name}.
func (h *Handler) DeleteSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSubnetParams) {
	logger := h.Logger.With("provider", "network", "resource", "subnet")
	id := &SubnetIdentity{name: name, tenant: tenant, workspace: workspace, network: network}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.SubnetWriter, newSubnetWithIdentity))
}

// GetSubnet handles GET /v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/subnets/{name}.
func (h *Handler) GetSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "subnet")
	ir := &SubnetIdentity{name: name, tenant: tenant, workspace: workspace, network: network}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.SubnetReader, newSubnetWithIdentity), subnetToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateSubnet handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/subnets/{name}.
func (h *Handler) CreateOrUpdateSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSubnetParams) {
	logger := h.Logger.With("provider", "network", "resource", "subnet")
	id := &SubnetIdentity{name: name, tenant: tenant, workspace: workspace, network: network}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Subnet, *subnetdom.Subnet, *sdkschema.Subnet]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.SubnetWriter),
		Updater: frest.UpdaterFromRepo(h.SubnetWriter),
		APIToDomain: func(sdk sdkschema.Subnet, p persistencepkg.IdentifiableResource) (*subnetdom.Subnet, error) {
			return subnetFromAPI(sdk, p.(*SubnetIdentity), region)
		},
		DomainToAPI: subnetToAPIWithVerb(http.MethodPut),
	})
}
