package rest

import (
	"net/http"
	"strconv"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// ListInstances handles GET /v1/tenants/{tenant}/workspaces/{workspace}/instances.
func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdkcompute.ListInstancesParams) {
	logger := h.Logger.With("provider", "compute", "resource", "instance")
	frest.HandleList(w, r, logger, instanceListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.InstanceReader), instanceIteratorToAPI)
}

// DeleteInstance handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}.
func (h *Handler) DeleteInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.DeleteInstanceParams) {
	logger := h.Logger.With("provider", "compute", "resource", "instance", "name", name)
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.InstanceWriter, newInstanceWithIdentity))
}

// GetInstance handles GET /v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}.
func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "compute", "resource", "instance", "name", name)
	ir := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.InstanceReader, newInstanceWithIdentity), instanceToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateInstance handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}.
func (h *Handler) CreateOrUpdateInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.CreateOrUpdateInstanceParams) {
	logger := h.Logger.With("provider", "compute", "resource", "instance", "name", name)
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Instance, *instancedom.Instance, *sdkschema.Instance]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.InstanceWriter),
		Updater: frest.UpdaterFromRepo(h.InstanceWriter),
		APIToDomain: func(sdk sdkschema.Instance, p persistencepkg.IdentifiableResource) *instancedom.Instance {
			return instanceFromAPI(sdk, p.(*resource.Identity), region)
		},
		DomainToAPI: instanceToAPIWithVerb(http.MethodPut),
	})
}
