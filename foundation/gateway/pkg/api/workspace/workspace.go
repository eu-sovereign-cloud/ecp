package workspace

import (
	"fmt"
	"strconv"

	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	v1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/status"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

func ListParamsFromAPI(params sdkworkspace.ListWorkspacesParams, tenant string) model.ListParams {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	return model.ListParams{
		Scope: scope.Scope{
			Tenant: tenant,
		},
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
}

func DomainToAPI(domain *regional.WorkspaceDomain) schema.Workspace {
	return mapWorkspaceDomainToAPI(*domain, "get")
}

func APIToDomain(api schema.Workspace, params port.IdentifiableResource) *regional.WorkspaceDomain {
	return &regional.WorkspaceDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name:            params.GetName(),
				ResourceVersion: params.GetVersion(),
			},
			Scope: scope.Scope{
				Tenant:    params.GetTenant(),
				Workspace: params.GetName(),
			},
			Annotations: api.Annotations,
			Labels:      api.Labels,
			Extensions:  api.Extensions,
		},
		Spec: api.Spec,
	}
}

func DomainToAPIIterator(domainWorkspaces []*regional.WorkspaceDomain, nextSkipToken *string) *sdkworkspace.WorkspaceIterator {
	sdkWorkspaces := make([]schema.Workspace, len(domainWorkspaces))
	for i, dom := range domainWorkspaces {
		sdkWorkspaces[i] = mapWorkspaceDomainToAPI(*dom, "list")
	}

	iterator := &sdkworkspace.WorkspaceIterator{
		Items: sdkWorkspaces,
		Metadata: schema.ResponseMetadata{
			Provider: regional.ProviderWorkspaceName,
			Resource: workspacev1.Resource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// mapWorkspaceDomainToAPI maps a WorkspaceDomain to schema.Workspace API object.
func mapWorkspaceDomainToAPI(domain regional.WorkspaceDomain, verb string) schema.Workspace {
	resVersion := 0
	// resourceVersion is best-effort numeric
	if rv, err := strconv.Atoi(domain.ResourceVersion); err == nil {
		resVersion = rv
	}

	refObj := schema.ReferenceObject{
		Resource: fmt.Sprintf(regional.ResourceFormat, schema.RegionalResourceMetadataKindResourceKindWorkspace, domain.Name),
		Provider: &domain.Provider,
		Region:   &domain.Region,
		Tenant:   &domain.Tenant,
	}
	ref := schema.Reference{}
	_ = ref.FromReferenceObject(refObj) // ignore mapping error, not critical internally

	var resourceState *schema.ResourceState
	if domain.Status != nil && domain.Status.State != nil {
		rs := status.MapResourceStateDomainToAPI(*domain.Status.State)
		resourceState = &rs
	}
	sdk := schema.Workspace{
		Spec: domain.Spec,
		Metadata: &schema.RegionalResourceMetadata{
			ApiVersion:      v1.Version,
			CreatedAt:       domain.CreatedAt,
			LastModifiedAt:  domain.UpdatedAt,
			Kind:            schema.RegionalResourceMetadataKindResourceKindWorkspace,
			Name:            domain.Name,
			Tenant:          domain.Tenant,
			Provider:        domain.Provider,
			Region:          domain.Region,
			Resource:        fmt.Sprintf(regional.TenantScopedResourceFormat, domain.Tenant, schema.RegionalResourceMetadataKindResourceKindWorkspace, domain.Name),
			Ref:             &ref,
			ResourceVersion: resVersion,
			Verb:            verb,
		},
		Labels:      domain.Labels,
		Annotations: domain.Annotations,
		Extensions:  domain.Extensions,
	}
	if domain.Status != nil {
		sdk.Status = &schema.WorkspaceStatus{
			ResourceCount: domain.Status.ResourceCount,
			State:         resourceState,
			Conditions:    status.MapConditionDomainsToAPI(domain.Status.Conditions),
		}
	}
	if domain.DeletedAt != nil {
		sdk.Metadata.DeletedAt = domain.DeletedAt
	}
	return sdk
}
