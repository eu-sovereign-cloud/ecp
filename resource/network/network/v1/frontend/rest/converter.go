// Package rest provides REST↔domain conversion and HTTP handlers for the network resource.
package rest

import (
	"fmt"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/network/v1"
)

const (
	// NetworkAPIVersion is the API version string used in response metadata.
	NetworkAPIVersion = netdom.Version
	// NetworkResource is the resource name.
	NetworkResource = netdom.Resource
	// ResourceFormat formats a resource path string.
	ResourceFormat = "%s/%s"
	// WorkspaceScopedResourceFormat formats a workspace-scoped resource ref.
	WorkspaceScopedResourceFormat = "tenants/%s/workspaces/%s/providers/%s/%s"
)

// NetworkIdentity carries identity for a single network resource.
type NetworkIdentity struct {
	name            string
	tenant          string
	workspace       string
	resourceVersion string
}

func (n *NetworkIdentity) GetName() string      { return n.name }
func (n *NetworkIdentity) GetVersion() string   { return n.resourceVersion }
func (n *NetworkIdentity) GetTenant() string    { return n.tenant }
func (n *NetworkIdentity) GetWorkspace() string { return n.workspace }

var _ persistence.IdentifiableResource = (*NetworkIdentity)(nil)

// ListParamsFromAPI converts SDK ListNetworksParams to resource.ListParams.
func ListParamsFromAPI(params sdknetwork.ListNetworksParams, tenant, workspace string) resource.ListParams {
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
			Tenant:    tenant,
			Workspace: workspace,
		},
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// NetworkDomainToAPIWithVerb returns a func that converts a Network to its SDK representation with the given verb.
func NetworkDomainToAPIWithVerb(verb string) func(domain *netdom.Network) *sdkschema.Network {
	return func(domain *netdom.Network) *sdkschema.Network {
		sdk := networkDomainToAPI(domain)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// networkDomainToAPI converts a Network to its SDK representation.
func networkDomainToAPI(domain *netdom.Network) *sdkschema.Network {
	resVersion := int64(0)
	if rv, err := strconv.ParseInt(domain.ResourceVersion, 10, 64); err == nil {
		resVersion = rv
	}

	n := &sdkschema.Network{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:     NetworkAPIVersion,
			CreatedAt:      domain.CreatedAt,
			LastModifiedAt: domain.UpdatedAt,
			Kind:           sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindNetwork,
			Name:           domain.Name,
			Tenant:         domain.Tenant,
			Workspace:      domain.Workspace,
			Provider:       domain.Provider,
			Region:         domain.Region,
			Resource:       fmt.Sprintf(ResourceFormat, sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindNetwork, domain.Name),
			Ref: fmt.Sprintf(
				domain.Provider+"/"+WorkspaceScopedResourceFormat,
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
			SkuRef:        commonfrontend.ReferenceToAPI(domain.Spec.SkuRef),
			RouteTableRef: commonfrontend.ReferenceToAPI(domain.Spec.RouteTableRef),
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
			Conditions: commonfrontend.ConditionsToAPI(domain.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(domain.Status.State),
		}
	}
	if domain.DeletedAt != nil {
		n.Metadata.DeletedAt = domain.DeletedAt
	}
	return n
}

// NetworkDomainToAPIIterator converts a list of Network to an SDK NetworkIterator.
func NetworkDomainToAPIIterator(domains []*netdom.Network, nextSkipToken *string) *sdknetwork.NetworkIterator {
	items := make([]sdkschema.Network, len(domains))
	for i := range domains {
		items[i] = *networkDomainToAPI(domains[i])
	}

	iterator := &sdknetwork.NetworkIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: netdom.ProviderID,
			Resource: NetworkResource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// APIToNetworkDomain converts an SDK Network to a Network.
func APIToNetworkDomain(sdk sdkschema.Network, id *NetworkIdentity, region string) *netdom.Network {
	domain := &netdom.Network{
		Spec: netdom.NetworkSpec{
			Cidr:          cidrFromAPI(sdk.Spec.Cidr),
			SkuRef:        commonfrontend.ReferenceFromAPI(sdk.Spec.SkuRef),
			RouteTableRef: commonfrontend.ReferenceFromAPI(sdk.Spec.RouteTableRef),
		},
	}
	domain.Name = id.GetName()
	domain.ResourceVersion = id.GetVersion()
	domain.Provider = netdom.ProviderID
	domain.Tenant = id.GetTenant()
	domain.Workspace = id.GetWorkspace()
	domain.Region = region
	domain.Labels = sdk.Labels
	domain.Annotations = sdk.Annotations
	domain.Extensions = sdk.Extensions

	for _, c := range sdk.Spec.AdditionalCidrs {
		domain.Spec.AdditionalCidrs = append(domain.Spec.AdditionalCidrs, cidrFromAPI(c))
	}

	return domain
}

// cidrDomainToAPI converts a netdom.Cidr to an sdkschema.Cidr.
func cidrDomainToAPI(c netdom.Cidr) sdkschema.Cidr {
	return sdkschema.Cidr{
		Ipv4: c.IPv4,
		Ipv6: c.IPv6,
	}
}

// cidrFromAPI converts an sdkschema.Cidr to a netdom.Cidr.
func cidrFromAPI(c sdkschema.Cidr) netdom.Cidr {
	return netdom.Cidr{
		IPv4: c.Ipv4,
		IPv6: c.Ipv6,
	}
}
