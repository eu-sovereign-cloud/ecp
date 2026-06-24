// Package rest provides REST↔domain conversion and HTTP handlers for the workspace resource.
package rest

import (
	"fmt"
	"net/http"
	"strconv"

	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

const (
	// WorkspaceAPIVersion is the API version string used in response metadata.
	WorkspaceAPIVersion = wsdom.Version
	// WorkspaceResource is the resource name.
	WorkspaceResource = wsdom.Resource
	// ResourceFormat formats a resource path string.
	ResourceFormat = "%s/%s"
	// TenantScopedResourceFormat formats a tenant-scoped resource ref.
	TenantScopedResourceFormat = "tenants/%s/providers/%s/%s"
)

// WorkspaceIdentity carries identity for a single workspace resource.
type WorkspaceIdentity struct {
	name            string
	tenant          string
	resourceVersion string
}

func (w *WorkspaceIdentity) GetName() string      { return w.name }
func (w *WorkspaceIdentity) GetVersion() string   { return w.resourceVersion }
func (w *WorkspaceIdentity) GetTenant() string    { return w.tenant }
func (w *WorkspaceIdentity) GetWorkspace() string { return "" }

var _ persistence.IdentifiableResource = (*WorkspaceIdentity)(nil)

// ListParamsFromAPI converts SDK ListWorkspacesParams to resource.ListParams.
func ListParamsFromAPI(params sdkworkspace.ListWorkspacesParams, tenant string) resource.ListParams {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	return resource.ListParams{
		Scope: resource.Scope{
			Tenant: tenant,
		},
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// DomainToAPIWithVerb returns a func that converts a Workspace to its SDK representation with the given verb.
func DomainToAPIWithVerb(verb string) func(domain *wsdom.Workspace) *sdkschema.Workspace {
	return func(domain *wsdom.Workspace) *sdkschema.Workspace {
		return domainToAPI(*domain, verb)
	}
}

// DomainToAPIIterator converts a list of Workspace to an SDK WorkspaceIterator.
func DomainToAPIIterator(domains []*wsdom.Workspace, nextSkipToken *string) *sdkworkspace.WorkspaceIterator {
	items := make([]sdkschema.Workspace, len(domains))
	for i, dom := range domains {
		items[i] = *(domainToAPI(*dom, http.MethodGet))
	}

	iterator := &sdkworkspace.WorkspaceIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: wsdom.ProviderID,
			Resource: WorkspaceResource,
			Verb:     http.MethodGet,
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// domainToAPI converts a Workspace to a schema.Workspace with the given verb.
func domainToAPI(domain wsdom.Workspace, verb string) *sdkschema.Workspace {
	resVersion := int64(0)
	if rv, err := strconv.ParseInt(domain.ResourceVersion, 10, 64); err == nil {
		resVersion = rv
	}

	sdk := &sdkschema.Workspace{
		Metadata: &sdkschema.RegionalResourceMetadata{
			ApiVersion:      WorkspaceAPIVersion,
			CreatedAt:       domain.CreatedAt,
			LastModifiedAt:  domain.UpdatedAt,
			Kind:            sdkschema.RegionalResourceMetadataKindResourceKindWorkspace,
			Name:            domain.Name,
			Tenant:          domain.Tenant,
			Provider:        domain.Provider,
			Region:          domain.Region,
			Resource:        fmt.Sprintf(ResourceFormat, sdkschema.RegionalResourceMetadataKindResourceKindWorkspace, domain.Name),
			Ref:             fmt.Sprintf(domain.Provider+"/"+TenantScopedResourceFormat, domain.Tenant, sdkschema.RegionalResourceMetadataKindResourceKindWorkspace, domain.Name),
			ResourceVersion: resVersion,
			Verb:            verb,
		},
		Labels:      domain.Labels,
		Annotations: domain.Annotations,
		Extensions:  domain.Extensions,
		Spec:        domain.Spec,
	}
	if sdk.Labels == nil {
		sdk.Labels = make(sdkschema.Labels)
	}
	if domain.Status != nil {
		sdk.Status = &sdkschema.WorkspaceStatus{
			ResourceCount: domain.Status.ResourceCount,
			State:         commonfrontend.ResourceStateToAPI(domain.Status.State),
			Conditions:    commonfrontend.ConditionsToAPI(domain.Status.Conditions),
		}
	}
	if domain.DeletedAt != nil {
		sdk.Metadata.DeletedAt = domain.DeletedAt
	}
	return sdk
}

// APIToDomain converts an SDK Workspace to a Workspace.
func APIToDomain(api sdkschema.Workspace, id *WorkspaceIdentity, region string) *wsdom.Workspace {
	domain := &wsdom.Workspace{
		Spec: api.Spec,
	}
	domain.Name = id.GetName()
	domain.ResourceVersion = id.GetVersion()
	domain.Provider = wsdom.ProviderID
	domain.Tenant = id.GetTenant()
	domain.Region = region
	domain.Labels = api.Labels
	domain.Annotations = api.Annotations
	domain.Extensions = api.Extensions

	return domain
}
