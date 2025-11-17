package regionalhandler

import (
	"log/slog"
	"net/http"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/port"
)

type Network struct {
	provider port.NetworkProvider
	logger   *slog.Logger
}

var _ sdknetwork.ServerInterface = (*Network)(nil) // Ensure Network implements the sdknetwork.ServerInterface.

func NewNetwork(logger *slog.Logger) *Network {
	return &Network{logger: logger.With("component", "Network")}
}

func (n *Network) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdknetwork.ListSkusParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) ListInternetGateways(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListInternetGatewaysParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) DeleteInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteInternetGatewayParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) GetInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) CreateOrUpdateInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateInternetGatewayParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) ListNetworks(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListNetworksParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) DeleteNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteNetworkParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) GetNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) CreateOrUpdateNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateNetworkParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) ListRouteTables(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, params sdknetwork.ListRouteTablesParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) DeleteRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteRouteTableParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) GetRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) CreateOrUpdateRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateRouteTableParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) ListSubnets(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, params sdknetwork.ListSubnetsParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) DeleteSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSubnetParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) GetSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) CreateOrUpdateSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSubnetParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) ListNics(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListNicsParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) DeleteNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteNicParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) GetNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) CreateOrUpdateNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateNicParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) ListPublicIps(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListPublicIpsParams) {
	// TODO implement me
	// n.provider.ListPublicIps()
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) DeletePublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeletePublicIpParams) {
	// TODO implement me
	// n.provider.DeletePublicIp()
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) GetPublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	// n.provider.GetPublicIp()
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) CreateOrUpdatePublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdatePublicIpParams) {
	// TODO implement me
	// n.provider.CreateOrUpdatePublicIp()
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) ListSecurityGroups(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListSecurityGroupsParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) DeleteSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSecurityGroupParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) GetSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *Network) CreateOrUpdateSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSecurityGroupParams) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}
