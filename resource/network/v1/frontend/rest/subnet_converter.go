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
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

// SubnetIdentity carries identity for a single subnet resource.
type SubnetIdentity struct {
	name            string
	tenant          string
	workspace       string
	network         string
	resourceVersion string
}

func (s *SubnetIdentity) GetName() string      { return s.name }
func (s *SubnetIdentity) GetVersion() string   { return s.resourceVersion }
func (s *SubnetIdentity) GetTenant() string    { return s.tenant }
func (s *SubnetIdentity) GetWorkspace() string { return s.workspace }
func (s *SubnetIdentity) GetNetwork() string   { return s.network }

var _ persistence.IdentifiableResource = (*SubnetIdentity)(nil)

// subnetListParams extends resource.ListParams with the network dimension. It satisfies
// resource.ListFilter via the embedded ListParams, and persistence.NetworkScope via GetNetwork,
// so ReaderAdapter[T].List resolves the per-network namespace for subnet without the Network
// field living on the shared ListParams struct every other resource also uses.
type subnetListParams struct {
	resource.ListParams
	Network string
}

func (p subnetListParams) GetNetwork() string { return p.Network }

// subnetListParamsFromAPI converts SDK ListSubnetsParams to subnetListParams.
func subnetListParamsFromAPI(params sdknetwork.ListSubnetsParams, tenant, workspace, network string) subnetListParams {
	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}
	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}
	return subnetListParams{
		ListParams: resource.ListParams{
			Scope:     resource.Scope{Tenant: tenant, Workspace: workspace},
			Limit:     validation.GetLimit(params.Limit),
			SkipToken: skipToken,
			Selector:  selector,
		},
		Network: network,
	}
}

// subnetToAPIWithVerb returns a func that converts a Subnet to its SDK representation with the given verb.
func subnetToAPIWithVerb(verb string) func(s *subnetdom.Subnet) *sdkschema.Subnet {
	return func(s *subnetdom.Subnet) *sdkschema.Subnet {
		sdk := subnetToAPI(s)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

func subnetToAPI(s *subnetdom.Subnet) *sdkschema.Subnet {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(s.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	const kind = sdkschema.RegionalNetworkResourceMetadataKindResourceKindSubnet

	out := &sdkschema.Subnet{
		Metadata: &sdkschema.RegionalNetworkResourceMetadata{
			ApiVersion:      subnetdom.Version,
			CreatedAt:       s.CreatedAt,
			LastModifiedAt:  s.UpdatedAt,
			Kind:            kind,
			Name:            s.Name,
			Tenant:          s.Tenant,
			Workspace:       s.Workspace,
			Network:         s.Network,
			Provider:        s.Provider,
			Region:          s.Region,
			Resource:        commondomain.FormatRegionalNetworkScopedResource(s.Network, kind, s.Name),
			Ref:             commondomain.FormatRegionalNetworkScopedRef(s.Provider, s.Tenant, s.Workspace, s.Network, kind, s.Name),
			ResourceVersion: resourceVersion,
		},
		Labels:      s.Labels,
		Annotations: s.Annotations,
		Extensions:  s.Extensions,
		Spec: sdkschema.SubnetSpec{
			Cidr:          cidrToAPI(netdom.CIDR(s.Spec.Cidr)),
			RouteTableRef: commonfrontend.ReferenceToAPI(s.Spec.RouteTableRef),
			Zone:          s.Spec.Zone,
		},
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}
	if s.Spec.SkuRef != (commondomain.Reference{}) {
		ref := commonfrontend.ReferenceToAPI(s.Spec.SkuRef)
		out.Spec.SkuRef = &ref
	}

	if s.Status != nil {
		out.Status = &sdkschema.SubnetStatus{
			Conditions: commonfrontend.ConditionsToAPI(s.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(s.Status.State),
		}
		if s.Status.Cidr != nil {
			cidr := cidrToAPI(netdom.CIDR(*s.Status.Cidr))
			out.Status.Cidr = &cidr
		}
		if s.Status.RouteTableRef != nil {
			ref := commonfrontend.ReferenceToAPI(*s.Status.RouteTableRef)
			out.Status.RouteTableRef = &ref
		}
	}
	if s.DeletedAt != nil {
		out.Metadata.DeletedAt = s.DeletedAt
	}
	return out
}

// subnetIteratorToAPI converts a list of Subnet to an SDK SubnetIterator.
func subnetIteratorToAPI(ss []*subnetdom.Subnet, nextSkipToken *string) *sdknetwork.SubnetIterator {
	items := make([]sdkschema.Subnet, len(ss))
	for i := range ss {
		items[i] = *subnetToAPI(ss[i])
	}
	iterator := &sdknetwork.SubnetIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: subnetdom.ProviderID,
			Resource: subnetdom.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator
}

// subnetFromAPI converts an SDK Subnet to a Subnet.
func subnetFromAPI(sdk sdkschema.Subnet, id *SubnetIdentity, region string) *subnetdom.Subnet {
	s := &subnetdom.Subnet{
		Spec: subnetdom.SubnetSpec{
			Cidr:          subnetdom.CIDR(cidrFromAPI(sdk.Spec.Cidr)),
			RouteTableRef: commonfrontend.ReferenceFromAPI(sdk.Spec.RouteTableRef),
			Zone:          sdk.Spec.Zone,
		},
	}
	if sdk.Spec.SkuRef != nil {
		s.Spec.SkuRef = commonfrontend.ReferenceFromAPI(*sdk.Spec.SkuRef)
	}

	s.Name = id.GetName()
	s.ResourceVersion = id.GetVersion()
	s.Provider = subnetdom.ProviderID
	s.Tenant = id.GetTenant()
	s.Workspace = id.GetWorkspace()
	s.Network = id.GetNetwork()
	s.Region = region
	s.Labels = sdk.Labels
	s.Annotations = sdk.Annotations
	s.Extensions = sdk.Extensions

	return s
}

// newSubnetWithIdentity returns a *subnetdom.Subnet populated with identity fields from ir.
func newSubnetWithIdentity(ir persistence.IdentifiableResource) *subnetdom.Subnet {
	d := &subnetdom.Subnet{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	if id, ok := ir.(*SubnetIdentity); ok {
		d.Network = id.GetNetwork()
	}
	return d
}
