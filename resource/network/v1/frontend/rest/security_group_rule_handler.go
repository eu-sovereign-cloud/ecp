package rest

import (
	"net/http"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

// ListSecurityGroupRules handles GET /v1/tenants/{tenant}/workspaces/{workspace}/security-group-rules.
func (h *Handler) ListSecurityGroupRules(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListSecurityGroupRulesParams) {
	logger := h.Logger.With("provider", "network", "resource", "security-group-rule")
	frest.HandleList(w, r, logger, securityGroupRuleListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.SecurityGroupRuleReader), securityGroupRuleIteratorToAPI)
}

// DeleteSecurityGroupRule handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/security-group-rules/{name}.
func (h *Handler) DeleteSecurityGroupRule(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSecurityGroupRuleParams) {
	logger := h.Logger.With("provider", "network", "resource", "security-group-rule", "name", name)
	id := &SecurityGroupRuleIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.SecurityGroupRuleWriter, newSecurityGroupRuleWithIdentity))
}

// GetSecurityGroupRule handles GET /v1/tenants/{tenant}/workspaces/{workspace}/security-group-rules/{name}.
func (h *Handler) GetSecurityGroupRule(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "security-group-rule", "name", name)
	ir := &SecurityGroupRuleIdentity{name: name, tenant: tenant, workspace: workspace}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.SecurityGroupRuleReader, newSecurityGroupRuleWithIdentity), securityGroupRuleToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateSecurityGroupRule handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/security-group-rules/{name}.
func (h *Handler) CreateOrUpdateSecurityGroupRule(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSecurityGroupRuleParams) {
	logger := h.Logger.With("provider", "network", "resource", "security-group-rule", "name", name)
	id := &SecurityGroupRuleIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.SecurityGroupRule, *securitygroupruledom.SecurityGroupRule, *sdkschema.SecurityGroupRule]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.SecurityGroupRuleWriter),
		Updater: frest.UpdaterFromRepo(h.SecurityGroupRuleWriter),
		APIToDomain: func(sdk sdkschema.SecurityGroupRule, p persistencepkg.IdentifiableResource) *securitygroupruledom.SecurityGroupRule {
			return securityGroupRuleFromAPI(sdk, p.(*SecurityGroupRuleIdentity), region)
		},
		DomainToAPI: securityGroupRuleToAPIWithVerb(http.MethodPut),
	})
}
