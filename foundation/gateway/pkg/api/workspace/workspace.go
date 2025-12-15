package workspace

import (
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	sdkv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

func ListParamsFromSDK(params sdkv1.ListWorkspacesParams, tenant string) model.ListParams {
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
		Tenant:    tenant,
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
}

func DomainToAPI(domain regional.WorkspaceDomain) schema.Workspace {
	return regional.MapWorkspaceDomainToAPI(domain, "get")
}

func DomainToAPIIterator(domainWorkspaces []regional.WorkspaceDomain, nextSkipToken *string) *sdkv1.WorkspaceIterator {
	sdkWorkspaces := make([]schema.Workspace, len(domainWorkspaces))
	for i, dom := range domainWorkspaces {
		sdkWorkspaces[i] = regional.MapWorkspaceDomainToAPI(dom, "list")
	}

	iterator := &sdkv1.WorkspaceIterator{
		Items: sdkWorkspaces,
		Metadata: schema.ResponseMetadata{
			Provider: regional.ProviderWorkspaceName, // TODO: dummy value, should actually retrieve provider name from cluster
			Resource: workspacev1.Resource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}
