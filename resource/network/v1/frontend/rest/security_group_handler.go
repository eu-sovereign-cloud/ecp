package rest

import (
	"net/http"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

// ListSecurityGroups handles GET /v1/tenants/{tenant}/workspaces/{workspace}/security-groups.
func (h *Handler) ListSecurityGroups(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListSecurityGroupsParams) {
	logger := h.Logger.With("provider", "network", "resource", "security-group")
	frest.HandleList(w, r, logger, securityGroupListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.SecurityGroupReader), securityGroupIteratorToAPI)
}

// DeleteSecurityGroup handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/security-groups/{name}.
func (h *Handler) DeleteSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSecurityGroupParams) {
	logger := h.Logger.With("provider", "network", "resource", "security-group")
	id := &SecurityGroupIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.SecurityGroupWriter, newSecurityGroupWithIdentity))
}

// GetSecurityGroup handles GET /v1/tenants/{tenant}/workspaces/{workspace}/security-groups/{name}.
func (h *Handler) GetSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "security-group")
	ir := &SecurityGroupIdentity{name: name, tenant: tenant, workspace: workspace}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.SecurityGroupReader, newSecurityGroupWithIdentity), securityGroupToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateSecurityGroup handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/security-groups/{name}.
func (h *Handler) CreateOrUpdateSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSecurityGroupParams) {
	logger := h.Logger.With("provider", "network", "resource", "security-group")
	id := &SecurityGroupIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.SecurityGroup, *securitygroupdom.SecurityGroup, *sdkschema.SecurityGroup]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.SecurityGroupWriter),
		Updater: frest.UpdaterFromRepo(h.SecurityGroupWriter),
		APIToDomain: func(sdk sdkschema.SecurityGroup, p persistencepkg.IdentifiableResource) (*securitygroupdom.SecurityGroup, error) {
			return securityGroupFromAPI(sdk, p.(*SecurityGroupIdentity), region)
		},
		DomainToAPI: securityGroupToAPIWithVerb(http.MethodPut),
	})
}
