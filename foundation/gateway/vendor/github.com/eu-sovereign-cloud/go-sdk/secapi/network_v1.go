package secapi

import (
	"context"
	"fmt"
	"net/http"

	network "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"k8s.io/utils/ptr"
)

type NetworkV1 struct {
	API
	network network.ClientWithResponsesInterface
}

// Network Sku

func (api *NetworkV1) ListSkus(ctx context.Context, tid TenantID) (*Iterator[schema.NetworkSku], error) {
	iter := Iterator[schema.NetworkSku]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.NetworkSku, *string, error) {
			resp, err := api.network.ListSkusWithResponse(ctx, schema.TenantPathParam(tid), &network.ListSkusParams{
				Accept: ptr.To(network.ListSkusParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *NetworkV1) GetSku(ctx context.Context, tref TenantReference) (*schema.NetworkSku, error) {
	if err := tref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.network.GetSkuWithResponse(ctx, schema.TenantPathParam(tref.Tenant), tref.Name, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

// Network

func (api *NetworkV1) ListNetworks(ctx context.Context, tid TenantID, wid WorkspaceID) (*Iterator[schema.Network], error) {
	iter := Iterator[schema.Network]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.Network, *string, error) {
			resp, err := api.network.ListNetworksWithResponse(ctx, schema.TenantPathParam(tid), schema.WorkspacePathParam(wid), &network.ListNetworksParams{
				Accept: ptr.To(network.ListNetworksParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *NetworkV1) GetNetwork(ctx context.Context, wref WorkspaceReference) (*schema.Network, error) {
	if err := wref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.network.GetNetworkWithResponse(ctx, schema.TenantPathParam(wref.Tenant), schema.WorkspacePathParam(wref.Workspace), wref.Name, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *NetworkV1) CreateOrUpdateNetworkWithParams(ctx context.Context, net *schema.Network, params *network.CreateOrUpdateNetworkParams) (*schema.Network, error) {
	if err := api.validateRegionalMetadata(net.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.network.CreateOrUpdateNetworkWithResponse(ctx, schema.TenantPathParam(net.Metadata.Tenant), schema.WorkspacePathParam(net.Metadata.Workspace), net.Metadata.Name, params, *net, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *NetworkV1) CreateOrUpdateNetwork(ctx context.Context, net *schema.Network) (*schema.Network, error) {
	return api.CreateOrUpdateNetworkWithParams(ctx, net, nil)
}

func (api *NetworkV1) DeleteNetworkWithParams(ctx context.Context, net *schema.Network, params *network.DeleteNetworkParams) error {
	if err := api.validateRegionalMetadata(net.Metadata); err != nil {
		return err
	}

	resp, err := api.network.DeleteNetworkWithResponse(ctx, net.Metadata.Tenant, net.Metadata.Workspace, net.Metadata.Name, params, api.loadRequestHeaders, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *NetworkV1) DeleteNetwork(ctx context.Context, net *schema.Network) error {
	return api.DeleteNetworkWithParams(ctx, net, nil)
}

// Subnet

func (api *NetworkV1) ListSubnets(ctx context.Context, tid TenantID, wid WorkspaceID, nid NetworkID) (*Iterator[schema.Subnet], error) {
	iter := Iterator[schema.Subnet]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.Subnet, *string, error) {
			resp, err := api.network.ListSubnetsWithResponse(ctx, schema.TenantPathParam(tid), schema.WorkspacePathParam(wid), schema.NetworkPathParam(nid), &network.ListSubnetsParams{
				Accept: ptr.To(network.ListSubnetsParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *NetworkV1) GetSubnet(ctx context.Context, nref NetworkReference) (*schema.Subnet, error) {
	if err := nref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.network.GetSubnetWithResponse(ctx, schema.TenantPathParam(nref.Tenant), schema.WorkspacePathParam(nref.Workspace), schema.NetworkPathParam(nref.Network), schema.ResourcePathParam(nref.Name), api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *NetworkV1) CreateOrUpdateSubnetWithParams(ctx context.Context, sub *schema.Subnet, params *network.CreateOrUpdateSubnetParams) (*schema.Subnet, error) {
	if err := api.validateNetworkMetadata(sub.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.network.CreateOrUpdateSubnetWithResponse(ctx, schema.TenantPathParam(sub.Metadata.Tenant), sub.Metadata.Workspace, sub.Metadata.Network, sub.Metadata.Name, params, *sub, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *NetworkV1) CreateOrUpdateSubnet(ctx context.Context, sub *schema.Subnet) (*schema.Subnet, error) {
	return api.CreateOrUpdateSubnetWithParams(ctx, sub, nil)
}

func (api *NetworkV1) DeleteSubnetWithParams(ctx context.Context, sub *schema.Subnet, params *network.DeleteSubnetParams) error {
	if err := api.validateNetworkMetadata(sub.Metadata); err != nil {
		return err
	}

	resp, err := api.network.DeleteSubnetWithResponse(ctx, sub.Metadata.Tenant, sub.Metadata.Workspace, sub.Metadata.Network, sub.Metadata.Name, params, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *NetworkV1) DeleteSubnet(ctx context.Context, sub *schema.Subnet) error {
	return api.DeleteSubnetWithParams(ctx, sub, nil)
}

// Route Table

func (api *NetworkV1) ListRouteTables(ctx context.Context, tid TenantID, wid WorkspaceID, nid NetworkID) (*Iterator[schema.RouteTable], error) {
	iter := Iterator[schema.RouteTable]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.RouteTable, *string, error) {
			resp, err := api.network.ListRouteTablesWithResponse(ctx, schema.TenantPathParam(tid), schema.WorkspacePathParam(wid), schema.NetworkPathParam(nid), &network.ListRouteTablesParams{
				Accept: ptr.To(network.ListRouteTablesParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *NetworkV1) GetRouteTable(ctx context.Context, nref NetworkReference) (*schema.RouteTable, error) {
	if err := nref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.network.GetRouteTableWithResponse(ctx, schema.TenantPathParam(nref.Tenant), schema.WorkspacePathParam(nref.Workspace), schema.NetworkPathParam(nref.Network), schema.ResourcePathParam(nref.Name), api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *NetworkV1) CreateOrUpdateRouteTableWithParams(ctx context.Context, route *schema.RouteTable, params *network.CreateOrUpdateRouteTableParams) (*schema.RouteTable, error) {
	if err := api.validateNetworkMetadata(route.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.network.CreateOrUpdateRouteTableWithResponse(ctx, schema.TenantPathParam(route.Metadata.Tenant), route.Metadata.Workspace, route.Metadata.Network, route.Metadata.Name, params, *route, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *NetworkV1) CreateOrUpdateRouteTable(ctx context.Context, route *schema.RouteTable) (*schema.RouteTable, error) {
	return api.CreateOrUpdateRouteTableWithParams(ctx, route, nil)
}

func (api *NetworkV1) DeleteRouteTableWithParams(ctx context.Context, route *schema.RouteTable, params *network.DeleteRouteTableParams) error {
	if err := api.validateNetworkMetadata(route.Metadata); err != nil {
		return err
	}

	resp, err := api.network.DeleteRouteTableWithResponse(ctx, route.Metadata.Tenant, route.Metadata.Workspace, route.Metadata.Network, route.Metadata.Name, params, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *NetworkV1) DeleteRouteTable(ctx context.Context, route *schema.RouteTable) error {
	return api.DeleteRouteTableWithParams(ctx, route, nil)
}

// Internet Gateway

func (api *NetworkV1) ListInternetGateways(ctx context.Context, tid TenantID, wid WorkspaceID) (*Iterator[schema.InternetGateway], error) {
	iter := Iterator[schema.InternetGateway]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.InternetGateway, *string, error) {
			resp, err := api.network.ListInternetGatewaysWithResponse(ctx, schema.TenantPathParam(tid), schema.WorkspacePathParam(wid), &network.ListInternetGatewaysParams{
				Accept: ptr.To(network.ListInternetGatewaysParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *NetworkV1) GetInternetGateway(ctx context.Context, wref WorkspaceReference) (*schema.InternetGateway, error) {
	if err := wref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.network.GetInternetGatewayWithResponse(ctx, schema.TenantPathParam(wref.Tenant), schema.WorkspacePathParam(wref.Workspace), wref.Name, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *NetworkV1) CreateOrUpdateInternetGatewayWithParams(ctx context.Context, gtw *schema.InternetGateway, params *network.CreateOrUpdateInternetGatewayParams) (*schema.InternetGateway, error) {
	if err := api.validateRegionalMetadata(gtw.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.network.CreateOrUpdateInternetGatewayWithResponse(ctx, schema.TenantPathParam(gtw.Metadata.Tenant), schema.WorkspacePathParam(gtw.Metadata.Workspace), gtw.Metadata.Name, params, *gtw, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *NetworkV1) CreateOrUpdateInternetGateway(ctx context.Context, gtw *schema.InternetGateway) (*schema.InternetGateway, error) {
	return api.CreateOrUpdateInternetGatewayWithParams(ctx, gtw, nil)
}

func (api *NetworkV1) DeleteInternetGatewayWithParams(ctx context.Context, gtw *schema.InternetGateway, params *network.DeleteInternetGatewayParams) error {
	if err := api.validateRegionalMetadata(gtw.Metadata); err != nil {
		return err
	}

	resp, err := api.network.DeleteInternetGatewayWithResponse(ctx, gtw.Metadata.Tenant, gtw.Metadata.Workspace, gtw.Metadata.Name, params, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *NetworkV1) DeleteInternetGateway(ctx context.Context, gtw *schema.InternetGateway) error {
	return api.DeleteInternetGatewayWithParams(ctx, gtw, nil)
}

// Security Group

func (api *NetworkV1) ListSecurityGroups(ctx context.Context, tid TenantID, wid WorkspaceID) (*Iterator[schema.SecurityGroup], error) {
	iter := Iterator[schema.SecurityGroup]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.SecurityGroup, *string, error) {
			resp, err := api.network.ListSecurityGroupsWithResponse(ctx, schema.TenantPathParam(tid), schema.WorkspacePathParam(wid), &network.ListSecurityGroupsParams{
				Accept: ptr.To(network.ListSecurityGroupsParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *NetworkV1) GetSecurityGroup(ctx context.Context, wref WorkspaceReference) (*schema.SecurityGroup, error) {
	if err := wref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.network.GetSecurityGroupWithResponse(ctx, schema.TenantPathParam(wref.Tenant), schema.WorkspacePathParam(wref.Workspace), wref.Name, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *NetworkV1) CreateOrUpdateSecurityGroupWithParams(ctx context.Context, group *schema.SecurityGroup, params *network.CreateOrUpdateSecurityGroupParams) (*schema.SecurityGroup, error) {
	if err := api.validateRegionalMetadata(group.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.network.CreateOrUpdateSecurityGroupWithResponse(ctx, schema.TenantPathParam(group.Metadata.Tenant), schema.WorkspacePathParam(group.Metadata.Workspace), group.Metadata.Name, params, *group, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *NetworkV1) CreateOrUpdateSecurityGroup(ctx context.Context, group *schema.SecurityGroup) (*schema.SecurityGroup, error) {
	return api.CreateOrUpdateSecurityGroupWithParams(ctx, group, nil)
}

func (api *NetworkV1) DeleteSecurityGroupWithParams(ctx context.Context, route *schema.SecurityGroup, params *network.DeleteSecurityGroupParams) error {
	if err := api.validateRegionalMetadata(route.Metadata); err != nil {
		return err
	}

	resp, err := api.network.DeleteSecurityGroupWithResponse(ctx, route.Metadata.Tenant, route.Metadata.Workspace, route.Metadata.Name, params, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *NetworkV1) DeleteSecurityGroup(ctx context.Context, route *schema.SecurityGroup) error {
	return api.DeleteSecurityGroupWithParams(ctx, route, nil)
}

// Nic

func (api *NetworkV1) ListNics(ctx context.Context, tid TenantID, wid WorkspaceID) (*Iterator[schema.Nic], error) {
	iter := Iterator[schema.Nic]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.Nic, *string, error) {
			resp, err := api.network.ListNicsWithResponse(ctx, schema.TenantPathParam(tid), schema.WorkspacePathParam(wid), &network.ListNicsParams{
				Accept: ptr.To(network.ListNicsParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *NetworkV1) GetNic(ctx context.Context, wref WorkspaceReference) (*schema.Nic, error) {
	if err := wref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.network.GetNicWithResponse(ctx, schema.TenantPathParam(wref.Tenant), schema.WorkspacePathParam(wref.Workspace), wref.Name, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *NetworkV1) CreateOrUpdateNicWithParams(ctx context.Context, nic *schema.Nic, params *network.CreateOrUpdateNicParams) (*schema.Nic, error) {
	if err := api.validateRegionalMetadata(nic.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.network.CreateOrUpdateNicWithResponse(ctx, schema.TenantPathParam(nic.Metadata.Tenant), schema.WorkspacePathParam(nic.Metadata.Workspace), nic.Metadata.Name, params, *nic, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *NetworkV1) CreateOrUpdateNic(ctx context.Context, nic *schema.Nic) (*schema.Nic, error) {
	return api.CreateOrUpdateNicWithParams(ctx, nic, nil)
}

func (api *NetworkV1) DeleteNicWithParams(ctx context.Context, nic *schema.Nic, params *network.DeleteNicParams) error {
	if err := api.validateRegionalMetadata(nic.Metadata); err != nil {
		return err
	}

	resp, err := api.network.DeleteNicWithResponse(ctx, nic.Metadata.Tenant, nic.Metadata.Workspace, nic.Metadata.Name, params, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *NetworkV1) DeleteNic(ctx context.Context, nic *schema.Nic) error {
	return api.DeleteNicWithParams(ctx, nic, nil)
}

// Public Ip

func (api *NetworkV1) ListPublicIps(ctx context.Context, tid TenantID, wid WorkspaceID) (*Iterator[schema.PublicIp], error) {
	iter := Iterator[schema.PublicIp]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.PublicIp, *string, error) {
			resp, err := api.network.ListPublicIpsWithResponse(ctx, schema.TenantPathParam(tid), schema.WorkspacePathParam(wid), &network.ListPublicIpsParams{
				Accept: ptr.To(network.ListPublicIpsParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *NetworkV1) GetPublicIp(ctx context.Context, wref WorkspaceReference) (*schema.PublicIp, error) {
	if err := wref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.network.GetPublicIpWithResponse(ctx, schema.TenantPathParam(wref.Tenant), schema.WorkspacePathParam(wref.Workspace), wref.Name, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *NetworkV1) CreateOrUpdatePublicIpWithParams(ctx context.Context, ip *schema.PublicIp, params *network.CreateOrUpdatePublicIpParams) (*schema.PublicIp, error) {
	if err := api.validateRegionalMetadata(ip.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.network.CreateOrUpdatePublicIpWithResponse(ctx, schema.TenantPathParam(ip.Metadata.Tenant), schema.WorkspacePathParam(ip.Metadata.Workspace), ip.Metadata.Name, params, *ip, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *NetworkV1) CreateOrUpdatePublicIp(ctx context.Context, ip *schema.PublicIp) (*schema.PublicIp, error) {
	return api.CreateOrUpdatePublicIpWithParams(ctx, ip, nil)
}

func (api *NetworkV1) DeletePublicIpWithParams(ctx context.Context, ip *schema.PublicIp, params *network.DeletePublicIpParams) error {
	if err := api.validateRegionalMetadata(ip.Metadata); err != nil {
		return err
	}

	resp, err := api.network.DeletePublicIpWithResponse(ctx, ip.Metadata.Tenant, ip.Metadata.Workspace, ip.Metadata.Name, params, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *NetworkV1) DeletePublicIp(ctx context.Context, ip *schema.PublicIp) error {
	return api.DeletePublicIpWithParams(ctx, ip, nil)
}

func (api *NetworkV1) BuildReferenceURN(urn string) (*schema.Reference, error) {
	urnRef := schema.ReferenceURN(urn)

	ref := &schema.Reference{}
	if err := ref.FromReferenceURN(urnRef); err != nil {
		return nil, fmt.Errorf("error building referenceURN from URN %s: %s", urn, err)
	}

	return ref, nil
}

func (api *NetworkV1) validateRegionalMetadata(metadata *schema.RegionalWorkspaceResourceMetadata) error {
	if metadata == nil {
		return ErrNoMetatada
	}

	if metadata.Tenant == "" {
		return ErrNoMetatadaTenant
	}

	if metadata.Workspace == "" {
		return ErrNoMetatadaWorkspace
	}

	return nil
}

func (api *NetworkV1) validateNetworkMetadata(metadata *schema.RegionalNetworkResourceMetadata) error {
	if metadata == nil {
		return ErrNoMetatada
	}

	if metadata.Tenant == "" {
		return ErrNoMetatadaTenant
	}

	if metadata.Workspace == "" {
		return ErrNoMetatadaWorkspace
	}

	return nil
}

func newNetworkV1(client *RegionalClient, networkUrl string) (*NetworkV1, error) {
	network, err := network.NewClientWithResponses(networkUrl)
	if err != nil {
		return nil, err
	}

	return &NetworkV1{API: API{authToken: client.authToken}, network: network}, nil
}
