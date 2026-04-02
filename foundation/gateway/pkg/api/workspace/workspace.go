package workspace

import (
	"fmt"
	"net/http"
	"strconv"

	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/regional/workspace/v1"
	v1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/regions/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/status"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/config"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional/consts"
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

func DomainToAPIWithVerb(verb string) func(domain *regional.WorkspaceDomain) *schema.Workspace {
	return func(domain *regional.WorkspaceDomain) *schema.Workspace {
		sdk := DomainToAPI(domain)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

func DomainToAPI(domain *regional.WorkspaceDomain) *schema.Workspace {
	return mapWorkspaceDomainToAPI(*domain, http.MethodGet)
}

func APIToDomain(api schema.Workspace, params port.IdentifiableResource) *regional.WorkspaceDomain {
	return &regional.WorkspaceDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name:            params.GetName(),
				ResourceVersion: params.GetVersion(),
				Provider:        consts.WorkspaceProvider,
			},
			Scope: scope.Scope{
				Tenant: params.GetTenant(),
			},
			Region:      config.Singleton().Region(),
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
		sdkWorkspaces[i] = *(mapWorkspaceDomainToAPI(*dom, http.MethodGet))
	}

	iterator := &sdkworkspace.WorkspaceIterator{
		Items: sdkWorkspaces,
		Metadata: schema.ResponseMetadata{
			Provider: consts.WorkspaceProvider,
			Resource: workspacev1.Resource,
			Verb:     http.MethodGet,
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// mapWorkspaceDomainToAPI maps a WorkspaceDomain to schema.Workspace API object.
func mapWorkspaceDomainToAPI(domain regional.WorkspaceDomain, verb string) *schema.Workspace {
	resVersion := int64(0)
	// resourceVersion is best-effort numeric
	if rv, err := strconv.ParseInt(domain.ResourceVersion, 10, 64); err == nil {
		resVersion = rv
	}

	sdk := &schema.Workspace{
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
			Ref:             fmt.Sprintf(regional.ResourceFormat, schema.RegionalResourceMetadataKindResourceKindWorkspace, domain.Name),
			ResourceVersion: resVersion,
			Verb:            verb,
		},
		Labels:      domain.Labels,
		Annotations: domain.Annotations,
		Extensions:  domain.Extensions,
		Spec:        domain.Spec,
	}
	// TODO: better solution to replace this workaround
	if sdk.Labels == nil {
		sdk.Labels = make(schema.Labels)
	}
	if domain.Status != nil {
		sdk.Status = &schema.WorkspaceStatus{
			ResourceCount: domain.Status.ResourceCount,
			State:         status.MapResourceStateDomainToAPI(domain.Status.State),
			Conditions:    status.MapConditionDomainsToAPI(domain.Status.Conditions),
		}
	}
	if domain.DeletedAt != nil {
		sdk.Metadata.DeletedAt = domain.DeletedAt
	}
	return sdk
}
