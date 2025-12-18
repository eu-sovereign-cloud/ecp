package workspace

import (
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
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
	return regional.MapWorkspaceDomainToAPI(*domain, "get")
}

func DomainToAPIIterator(domainWorkspaces []*regional.WorkspaceDomain, nextSkipToken *string) *sdkworkspace.WorkspaceIterator {
	sdkWorkspaces := make([]schema.Workspace, len(domainWorkspaces))
	for i, dom := range domainWorkspaces {
		sdkWorkspaces[i] = regional.MapWorkspaceDomainToAPI(*dom, "list")
	}

	iterator := &sdkworkspace.WorkspaceIterator{
		Items: sdkWorkspaces,
		Metadata: schema.ResponseMetadata{
			Provider: regional.ProviderWorkspaceName, // TODO: dummy value, should actually retrieve provider name from somewhere (cluster or maybe some runtime config)
			Resource: workspacev1.Resource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}
