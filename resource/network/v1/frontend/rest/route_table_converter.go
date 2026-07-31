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
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

// RouteTableIdentity carries identity for a single route table resource.
type RouteTableIdentity struct {
	name            string
	tenant          string
	workspace       string
	network         string
	resourceVersion string
}

func (rt *RouteTableIdentity) GetName() string      { return rt.name }
func (rt *RouteTableIdentity) GetVersion() string   { return rt.resourceVersion }
func (rt *RouteTableIdentity) GetTenant() string    { return rt.tenant }
func (rt *RouteTableIdentity) GetWorkspace() string { return rt.workspace }
func (rt *RouteTableIdentity) GetNetwork() string   { return rt.network }

var _ persistence.IdentifiableResource = (*RouteTableIdentity)(nil)

// routeTableListParams extends resource.ListParams with the network dimension. It satisfies
// resource.ListFilter via the embedded ListParams, and persistence.NetworkScope via GetNetwork,
// so ReaderAdapter[T].List resolves the per-network namespace for route-table without the
// Network field living on the shared ListParams struct every other resource also uses.
type routeTableListParams struct {
	resource.ListParams
	Network string
}

func (p routeTableListParams) GetNetwork() string { return p.Network }

// routeTableListParamsFromAPI converts SDK ListRouteTablesParams to routeTableListParams.
func routeTableListParamsFromAPI(params sdknetwork.ListRouteTablesParams, tenant, workspace, network string) routeTableListParams {
	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}
	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}
	return routeTableListParams{
		ListParams: resource.ListParams{
			Scope:     resource.Scope{Tenant: tenant, Workspace: workspace},
			Limit:     validation.GetLimit(params.Limit),
			SkipToken: skipToken,
			Selector:  selector,
		},
		Network: network,
	}
}

// routeTableToAPIWithVerb returns a func that converts a RouteTable to its SDK representation with the given verb.
func routeTableToAPIWithVerb(verb string) func(rt *routetabledom.RouteTable) *sdkschema.RouteTable {
	return func(rt *routetabledom.RouteTable) *sdkschema.RouteTable {
		sdk := routeTableToAPI(rt)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

func routeTableToAPI(rt *routetabledom.RouteTable) *sdkschema.RouteTable {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(rt.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	const kind = sdkschema.RegionalNetworkResourceMetadataKindResourceKindRoutingTable

	routes := make([]sdkschema.RouteSpec, len(rt.Spec.Routes))
	for i, route := range rt.Spec.Routes {
		routes[i] = sdkschema.RouteSpec{
			DestinationCidrBlock: route.DestinationCidrBlock,
			TargetRef:            commonfrontend.ReferenceToAPI(route.TargetRef),
		}
	}

	out := &sdkschema.RouteTable{
		Metadata: &sdkschema.RegionalNetworkResourceMetadata{
			ApiVersion:      routetabledom.Version,
			CreatedAt:       rt.CreatedAt,
			LastModifiedAt:  rt.UpdatedAt,
			Kind:            kind,
			Name:            rt.Name,
			Tenant:          rt.Tenant,
			Workspace:       rt.Workspace,
			Network:         rt.Network,
			Provider:        rt.Provider,
			Region:          rt.Region,
			Resource:        commondomain.FormatRegionalNetworkScopedResource(rt.Network, kind, rt.Name),
			Ref:             commondomain.FormatRegionalNetworkScopedRef(rt.Provider, rt.Tenant, rt.Workspace, rt.Network, kind, rt.Name),
			ResourceVersion: resourceVersion,
		},
		Labels:      rt.Labels,
		Annotations: rt.Annotations,
		Extensions:  rt.Extensions,
		Spec: sdkschema.RouteTableSpec{
			Routes: routes,
		},
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}

	if rt.Status != nil {
		routeStatuses := make([]sdkschema.RouteStatus, len(rt.Status.Routes))
		for i, rs := range rt.Status.Routes {
			routeStatuses[i] = sdkschema.RouteStatus{
				Conditions: commonfrontend.ConditionsToAPI(rs.Conditions),
				State:      commonfrontend.ResourceStateToAPI(rs.State),
			}
		}
		out.Status = &sdkschema.RouteTableStatus{
			Conditions: commonfrontend.ConditionsToAPI(rt.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(rt.Status.State),
			Routes:     routeStatuses,
		}
	}
	if rt.DeletedAt != nil {
		out.Metadata.DeletedAt = rt.DeletedAt
	}
	return out
}

// routeTableIteratorToAPI converts a list of RouteTable to an SDK RouteTableIterator.
func routeTableIteratorToAPI(rts []*routetabledom.RouteTable, nextSkipToken *string) *sdknetwork.RouteTableIterator {
	items := make([]sdkschema.RouteTable, len(rts))
	for i := range rts {
		items[i] = *routeTableToAPI(rts[i])
	}
	iterator := &sdknetwork.RouteTableIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: routetabledom.ProviderID,
			Resource: routetabledom.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator
}

// routeTableFromAPI converts an SDK RouteTable to a RouteTable.
func routeTableFromAPI(sdk sdkschema.RouteTable, id *RouteTableIdentity, region string) *routetabledom.RouteTable {
	routes := make([]routetabledom.RouteSpec, len(sdk.Spec.Routes))
	for i, route := range sdk.Spec.Routes {
		routes[i] = routetabledom.RouteSpec{
			DestinationCidrBlock: route.DestinationCidrBlock,
			TargetRef:            commonfrontend.ReferenceFromAPI(route.TargetRef),
		}
	}

	rt := &routetabledom.RouteTable{
		Spec: routetabledom.RouteTableSpec{
			Routes: routes,
		},
	}
	rt.Name = id.GetName()
	rt.ResourceVersion = id.GetVersion()
	rt.Provider = routetabledom.ProviderID
	rt.Tenant = id.GetTenant()
	rt.Workspace = id.GetWorkspace()
	rt.Network = id.GetNetwork()
	rt.Region = region
	rt.Labels = sdk.Labels
	rt.Annotations = sdk.Annotations
	rt.Extensions = sdk.Extensions

	return rt
}

// newRouteTableWithIdentity returns a *routetabledom.RouteTable populated with identity fields from ir.
func newRouteTableWithIdentity(ir persistence.IdentifiableResource) *routetabledom.RouteTable {
	d := &routetabledom.RouteTable{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	if id, ok := ir.(*RouteTableIdentity); ok {
		d.Network = id.GetNetwork()
	}
	return d
}
