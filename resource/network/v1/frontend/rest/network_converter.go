package rest

import (
	"fmt"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

const (
	// NetworkAPIVersion is the API version string used in response metadata.
	NetworkAPIVersion = netdom.Version
	// NetworkResource is the resource name.
	NetworkResource = netdom.Resource
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

// networkListParamsFromAPI converts SDK ListNetworksParams to resource.ListParams.
func networkListParamsFromAPI(params sdknetwork.ListNetworksParams, tenant, workspace string) resource.ListParams {
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

// networkToAPIWithVerb returns a func that converts a Network to its SDK representation with the given verb.
func networkToAPIWithVerb(verb string) func(n *netdom.Network) *sdkschema.Network {
	return func(n *netdom.Network) *sdkschema.Network {
		sdk := networkToAPI(n)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// networkToAPI converts a Network to its SDK representation.
func networkToAPI(n *netdom.Network) *sdkschema.Network {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(n.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	out := &sdkschema.Network{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:     NetworkAPIVersion,
			CreatedAt:      n.CreatedAt,
			LastModifiedAt: n.UpdatedAt,
			Kind:           sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindNetwork,
			Name:           n.Name,
			Tenant:         n.Tenant,
			Workspace:      n.Workspace,
			Provider:       n.Provider,
			Region:         n.Region,
			Resource:       fmt.Sprintf(commondomain.RegionalResourceFormat, sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindNetwork, n.Name),
			Ref: fmt.Sprintf(
				n.Provider+"/"+commondomain.RegionalWorkspaceScopedResourceFormat,
				n.Tenant,
				n.Workspace,
				sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindNetwork,
				n.Name,
			),
			ResourceVersion: resourceVersion,
		},
		Labels:      n.Labels,
		Annotations: n.Annotations,
		Extensions:  n.Extensions,
		Spec: sdkschema.NetworkSpec{
			Cidr:          cidrToAPI(n.Spec.CIDR),
			SkuRef:        commonfrontend.ReferenceToAPI(n.Spec.SkuRef),
			RouteTableRef: commonfrontend.ReferenceToAPI(n.Spec.RouteTableRef),
		},
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}

	for _, c := range n.Spec.AdditionalCIDRs {
		out.Spec.AdditionalCidrs = append(out.Spec.AdditionalCidrs, cidrToAPI(c))
	}

	if n.Status != nil {
		out.Status = &sdkschema.NetworkStatus{
			Conditions: commonfrontend.ConditionsToAPI(n.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(n.Status.State),
		}
	}
	if n.DeletedAt != nil {
		out.Metadata.DeletedAt = n.DeletedAt
	}
	return out
}

// networkIteratorToAPI converts a list of Network to an SDK NetworkIterator.
func networkIteratorToAPI(ns []*netdom.Network, nextSkipToken *string) *sdknetwork.NetworkIterator {
	items := make([]sdkschema.Network, len(ns))
	for i := range ns {
		items[i] = *networkToAPI(ns[i])
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

// networkFromAPI converts an SDK Network to a Network.
func networkFromAPI(sdk sdkschema.Network, id *NetworkIdentity, region string) *netdom.Network {
	n := &netdom.Network{
		Spec: netdom.NetworkSpec{
			CIDR:          cidrFromAPI(sdk.Spec.Cidr),
			SkuRef:        commonfrontend.ReferenceFromAPI(sdk.Spec.SkuRef),
			RouteTableRef: commonfrontend.ReferenceFromAPI(sdk.Spec.RouteTableRef),
		},
	}
	n.Name = id.GetName()
	n.ResourceVersion = id.GetVersion()
	n.Provider = netdom.ProviderID
	n.Tenant = id.GetTenant()
	n.Workspace = id.GetWorkspace()
	n.Region = region
	n.Labels = sdk.Labels
	n.Annotations = sdk.Annotations
	n.Extensions = sdk.Extensions

	for _, c := range sdk.Spec.AdditionalCidrs {
		n.Spec.AdditionalCIDRs = append(n.Spec.AdditionalCIDRs, cidrFromAPI(c))
	}

	return n
}

// cidrToAPI converts a netdom.CIDR to an sdkschema.Cidr.
func cidrToAPI(c netdom.CIDR) sdkschema.Cidr {
	return sdkschema.Cidr{
		Ipv4: c.IPv4,
		Ipv6: c.IPv6,
	}
}

// cidrFromAPI converts an sdkschema.Cidr to a netdom.CIDR.
func cidrFromAPI(c sdkschema.Cidr) netdom.CIDR {
	return netdom.CIDR{
		IPv4: c.Ipv4,
		IPv6: c.Ipv6,
	}
}
