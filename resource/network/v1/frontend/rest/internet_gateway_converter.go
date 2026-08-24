package rest

import (
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// InternetGatewayIdentity carries identity for a single internet gateway resource.
type InternetGatewayIdentity struct {
	name            string
	tenant          string
	workspace       string
	resourceVersion string
}

func (ig *InternetGatewayIdentity) GetName() string      { return ig.name }
func (ig *InternetGatewayIdentity) GetVersion() string   { return ig.resourceVersion }
func (ig *InternetGatewayIdentity) GetTenant() string    { return ig.tenant }
func (ig *InternetGatewayIdentity) GetWorkspace() string { return ig.workspace }

var _ persistence.IdentifiableResource = (*InternetGatewayIdentity)(nil)

// internetGatewayListParamsFromAPI converts SDK ListInternetGatewaysParams to resource.ListParams.
func internetGatewayListParamsFromAPI(params sdknetwork.ListInternetGatewaysParams, tenant, workspace string) resource.ListParams {
	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}
	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}
	return resource.ListParams{
		Scope:     resource.Scope{Tenant: tenant, Workspace: workspace},
		Limit:     validation.GetLimit(params.Limit),
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// internetGatewayToAPIWithVerb returns a func that converts an InternetGateway to its SDK representation with the given verb.
func internetGatewayToAPIWithVerb(verb string) func(ig *internetgatewaydom.InternetGateway) *sdkschema.InternetGateway {
	return func(ig *internetgatewaydom.InternetGateway) *sdkschema.InternetGateway {
		sdk := internetGatewayToAPI(ig)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

func internetGatewayToAPI(ig *internetgatewaydom.InternetGateway) *sdkschema.InternetGateway {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(ig.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	const kind = sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindInternetGateway

	out := &sdkschema.InternetGateway{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:      internetgatewaydom.Version,
			CreatedAt:       ig.CreatedAt,
			LastModifiedAt:  ig.UpdatedAt,
			Kind:            kind,
			Name:            ig.Name,
			Tenant:          ig.Tenant,
			Workspace:       ig.Workspace,
			Provider:        ig.Provider,
			Region:          ig.Region,
			Resource:        commondomain.FormatRegionalResource(kind, ig.Name),
			Ref:             commondomain.FormatRegionalWorkspaceScopedRef(ig.Provider, ig.Tenant, ig.Workspace, kind, ig.Name),
			ResourceVersion: resourceVersion,
		},
		Labels:      ig.Labels,
		Annotations: ig.Annotations,
		Extensions:  ig.Extensions,
		Spec: sdkschema.InternetGatewaySpec{
			EgressOnly: ig.Spec.EgressOnly,
		},
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}

	if ig.Status != nil {
		out.Status = &sdkschema.InternetGatewayStatus{
			Conditions: commonfrontend.ConditionsToAPI(ig.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(ig.Status.State),
		}
	}
	if ig.DeletedAt != nil {
		out.Metadata.DeletedAt = ig.DeletedAt
	}
	return out
}

// internetGatewayIteratorToAPI converts a list of InternetGateway to an SDK InternetGatewayIterator.
func internetGatewayIteratorToAPI(igs []*internetgatewaydom.InternetGateway, nextSkipToken *string) *sdknetwork.InternetGatewayIterator {
	items := make([]sdkschema.InternetGateway, len(igs))
	for i := range igs {
		items[i] = *internetGatewayToAPI(igs[i])
	}
	iterator := &sdknetwork.InternetGatewayIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: internetgatewaydom.ProviderID,
			Resource: internetgatewaydom.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator
}

// internetGatewayFromAPI converts an SDK InternetGateway to an InternetGateway.
func internetGatewayFromAPI(sdk sdkschema.InternetGateway, id *InternetGatewayIdentity, region string) (*internetgatewaydom.InternetGateway, error) {
	ig := &internetgatewaydom.InternetGateway{
		Spec: internetgatewaydom.InternetGatewaySpec{
			EgressOnly: sdk.Spec.EgressOnly,
		},
	}
	ig.Name = id.GetName()
	ig.ResourceVersion = id.GetVersion()
	ig.Provider = internetgatewaydom.ProviderID
	ig.Tenant = id.GetTenant()
	ig.Workspace = id.GetWorkspace()
	ig.Region = region
	ig.Labels = sdk.Labels
	ig.Annotations = sdk.Annotations
	ig.Extensions = sdk.Extensions

	return ig, nil
}

// newInternetGatewayWithIdentity returns a *internetgatewaydom.InternetGateway populated with identity fields from ir.
func newInternetGatewayWithIdentity(ir persistence.IdentifiableResource) *internetgatewaydom.InternetGateway {
	d := &internetgatewaydom.InternetGateway{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	return d
}
