package network

import (
	"fmt"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	networkv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network"
	networksv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network/networks/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/reference"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/config"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/converters/rest2domain/status"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/converters/rest2domain/validation"
	model "github.com/eu-sovereign-cloud/ecp/foundation/models/domain"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional/consts"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

// NetworkDomainToAPIWithVerb returns a func that converts a NetworkDomain to its SDK representation with the given verb.
func NetworkDomainToAPIWithVerb(verb string) func(domain *regional.NetworkDomain) *sdkschema.Network {
	return func(domain *regional.NetworkDomain) *sdkschema.Network {
		sdk := networkDomainToAPI(domain)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// networkDomainToAPI converts a NetworkDomain to its SDK representation.
func networkDomainToAPI(domain *regional.NetworkDomain) *sdkschema.Network {
	resVersion := int64(0)
	if rv, err := strconv.ParseInt(domain.ResourceVersion, 10, 64); err == nil {
		resVersion = rv
	}

	n := &sdkschema.Network{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:     networkv1.Version,
			CreatedAt:      domain.CreatedAt,
			LastModifiedAt: domain.UpdatedAt,
			Kind:           sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindNetwork,
			Name:           domain.Name,
			Tenant:         domain.Tenant,
			Workspace:      domain.Workspace,
			Provider:       domain.Provider,
			Region:         domain.Region,
			Resource:       fmt.Sprintf(regional.ResourceFormat, sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindNetwork, domain.Name),
			Ref: fmt.Sprintf(
				domain.Provider+"/"+regional.WorkspaceScopedResourceFormat,
				domain.Tenant,
				domain.Workspace,
				sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindNetwork,
				domain.Name,
			),
			ResourceVersion: resVersion,
		},
		Labels:      domain.Labels,
		Annotations: domain.Annotations,
		Extensions:  domain.Extensions,
		Spec: sdkschema.NetworkSpec{
			Cidr:          cidrDomainToAPI(domain.Spec.Cidr),
			SkuRef:        reference.ToAPI(domain.Spec.SkuRef),
			RouteTableRef: reference.ToAPI(domain.Spec.RouteTableRef),
		},
	}

	if n.Labels == nil {
		n.Labels = make(sdkschema.Labels)
	}

	for _, c := range domain.Spec.AdditionalCidrs {
		n.Spec.AdditionalCidrs = append(n.Spec.AdditionalCidrs, cidrDomainToAPI(c))
	}

	if domain.Status != nil {
		n.Status = &sdkschema.NetworkStatus{
			Conditions: status.ConditionDomainsToAPI(domain.Status.Conditions),
			State:      sdkschema.ResourceState(domain.Status.State),
		}
	}
	if domain.DeletedAt != nil {
		n.Metadata.DeletedAt = domain.DeletedAt
	}
	return n
}

// NetworkListParamsFromAPI converts SDK ListNetworksParams to model.ListParams.
func NetworkListParamsFromAPI(params sdknetwork.ListNetworksParams, tenant, workspace string) model.ListParams {
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
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
		Scope: scope.Scope{
			Tenant:    tenant,
			Workspace: workspace,
		},
	}
}

// NetworkDomainToAPIIterator converts a list of NetworkDomain to an SDK NetworkIterator.
func NetworkDomainToAPIIterator(domains []*regional.NetworkDomain, nextSkipToken *string) *sdknetwork.NetworkIterator {
	items := make([]sdkschema.Network, len(domains))
	for i := range domains {
		mapped := networkDomainToAPI(domains[i])
		items[i] = *mapped
	}

	iterator := &sdknetwork.NetworkIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: consts.NetworkProvider,
			Resource: networksv1.NetworkResource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// APIToNetworkDomain converts an SDK Network to a NetworkDomain.
func APIToNetworkDomain(sdk sdkschema.Network, params port.IdentifiableResource) *regional.NetworkDomain {
	domain := &regional.NetworkDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name:            params.GetName(),
				ResourceVersion: params.GetVersion(),
				Provider:        consts.NetworkProvider,
			},
			Scope: scope.Scope{
				Tenant:    params.GetTenant(),
				Workspace: params.GetWorkspace(),
			},
			Region:      config.Singleton().Region(),
			Labels:      sdk.Labels,
			Annotations: sdk.Annotations,
			Extensions:  sdk.Extensions,
		},
		Spec: regional.NetworkSpecDomain{
			Cidr:          cidrFromAPI(sdk.Spec.Cidr),
			SkuRef:        reference.FromAPI(sdk.Spec.SkuRef),
			RouteTableRef: reference.FromAPI(sdk.Spec.RouteTableRef),
		},
	}

	for _, c := range sdk.Spec.AdditionalCidrs {
		domain.Spec.AdditionalCidrs = append(domain.Spec.AdditionalCidrs, cidrFromAPI(c))
	}

	return domain
}

// cidrDomainToAPI converts a regional.CidrDomain to an sdkschema.Cidr.
func cidrDomainToAPI(c regional.CidrDomain) sdkschema.Cidr {
	return sdkschema.Cidr{
		Ipv4: c.IPv4,
		Ipv6: c.IPv6,
	}
}

// cidrFromAPI converts an sdkschema.Cidr to a regional.CidrDomain.
func cidrFromAPI(c sdkschema.Cidr) regional.CidrDomain {
	return regional.CidrDomain{
		IPv4: c.Ipv4,
		IPv6: c.Ipv6,
	}
}
