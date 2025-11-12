package regionalprovider

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/client-go/rest"

	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/network/skus/v1"
	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/provider/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
)

const (
	NetworkBaseURL      = "/providers/seca.network"
	ProviderNetworkName = "seca.network/v1"
)

type NetworkSKUsProvider interface {
	ListSKUs(ctx context.Context, tenantID string, params sdknetwork.ListSkusParams) (*sdknetwork.SkuIterator, error)
	GetSKU(ctx context.Context, tenantID, skuID string) (*sdkschema.NetworkSku, error)
}

type InternetGatewaysProvider interface {
	ListInternetGateways(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListInternetGatewaysParams) (*secapi.Iterator[sdkschema.InternetGateway], error)
	DeleteInternetGateway(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteInternetGatewayParams) error
	GetInternetGateway(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.InternetGateway, error)
	CreateOrUpdateInternetGateway(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateInternetGatewayParams, req sdkschema.InternetGateway) (*sdkschema.InternetGateway, bool, error)
}

type NetworksProvider interface {
	ListNetworks(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListNetworksParams) (*secapi.Iterator[sdkschema.Network], error)
	DeleteNetwork(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteNetworkParams) error
	GetNetwork(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.Network, error)
	CreateOrUpdateNetwork(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateNetworkParams, req sdkschema.Network) (*sdkschema.Network, bool, error)
}

type RouteTablesProvider interface {
	ListRouteTables(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListRouteTablesParams) (*secapi.Iterator[sdkschema.RouteTable], error)
	DeleteRouteTable(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteRouteTableParams) error
	GetRouteTable(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.RouteTable, error)
	CreateOrUpdateRouteTable(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateRouteTableParams, req sdkschema.RouteTable) (*sdkschema.RouteTable, bool, error)
}

type SubnetsProvider interface {
	ListSubnets(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListSubnetsParams) (*secapi.Iterator[sdkschema.Subnet], error)
	DeleteSubnet(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteSubnetParams) error
	GetSubnet(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.Subnet, error)
	CreateOrUpdateSubnet(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateSubnetParams, req sdkschema.Subnet) (*sdkschema.Subnet, bool, error)
}

type NicsProvider interface {
	ListNics(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListNicsParams) (*secapi.Iterator[sdkschema.Nic], error)
	DeleteNic(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteNicParams) error
	GetNic(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.Nic, error)
	CreateOrUpdateNic(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateNicParams, req sdkschema.Nic) (*sdkschema.Nic, bool, error)
}

type PublicIPsProvider interface {
	ListPublicIps(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListPublicIpsParams) (*secapi.Iterator[sdkschema.PublicIp], error)
	GetPublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string) (sdkschema.PublicIp, error)
	CreateOrUpdatePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params sdknetwork.CreateOrUpdatePublicIpParams, req sdkschema.PublicIp) (*sdkschema.PublicIp, bool, error)
	DeletePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params sdknetwork.DeletePublicIpParams) error
}

type SecurityGroupsProvider interface {
	ListSecurityGroups(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListSecurityGroupsParams) (*secapi.Iterator[sdkschema.SecurityGroup], error)
	DeleteSecurityGroup(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteSecurityGroupParams) error
	GetSecurityGroup(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.SecurityGroup, error)
	CreateOrUpdateSecurityGroup(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateSecurityGroupParams, req sdkschema.SecurityGroup) (*sdkschema.SecurityGroup, bool, error)
}

type NetworkProvider interface {
	NetworkSKUsProvider
	InternetGatewaysProvider
	NetworksProvider
	RouteTablesProvider
	SubnetsProvider
	NicsProvider
	PublicIPsProvider
	SecurityGroupsProvider
}

// NetworkController implements the NetworkProvider interface and provides methods to interact with the Network CRDs and XRDs in the Kubernetes cluster.
type NetworkController struct {
	client *kubeclient.KubeClient
	logger *slog.Logger
}

var _ NetworkProvider = (*NetworkController)(nil) // Ensure NetworkController implements the NetworkProvider interface.

// NewNetworkController creates a new NetworkController with a Kubernetes client.
func NewNetworkController(logger *slog.Logger, cfg *rest.Config) (*NetworkController, error) {
	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		logger.Error("failed to create kubeclient", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create kubeclient: %w", err)
	}

	return &NetworkController{
		client: client,
		logger: logger.With(slog.String("component", "NetworkController")),
	}, nil
}

func (c *NetworkController) ListSKUs(ctx context.Context, tenantID string, params sdknetwork.ListSkusParams) (
	*sdknetwork.SkuIterator, error,
) {
	limit := validation.GetLimit(params.Limit)

	convert := common.Adapter(func(crdNetworkSKU skuv1.NetworkSKU) (sdkschema.NetworkSku, error) {
		return fromCRToSDKNetworkSKU(crdNetworkSKU), nil
	})
	opts := common.NewListOptions().Namespace(tenantID)
	if limit > 0 {
		opts.Limit(limit)
	}
	if params.SkipToken != nil {
		opts.SkipToken(*params.SkipToken)
	}
	if params.Labels != nil {
		opts.Selector(*params.Labels)
	}

	sdkNetworkSKUs, nextSkipToken, err := common.ListResources(ctx, c.client.Client, skuv1.NetworkSKUGVR, *c.logger, convert, opts)
	if err != nil {
		return nil, err
	}

	iterator := sdknetwork.SkuIterator{
		Items: sdkNetworkSKUs,
		Metadata: sdkschema.ResponseMetadata{
			Provider: ProviderNetworkName,
			Resource: skuv1.NetworkSKUResource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return &iterator, nil
}

func (c *NetworkController) GetSKU(
	ctx context.Context, tenantID, skuID string,
) (*sdkschema.NetworkSku, error) {
	convert := common.Adapter(func(crdNetworkSKU skuv1.NetworkSKU) (sdkschema.NetworkSku, error) {
		return fromCRToSDKNetworkSKU(crdNetworkSKU), nil
	})
	opts := common.NewGetOptions().Namespace(tenantID)
	sku, err := common.GetResource(ctx, c.client.Client, skuv1.NetworkSKUGVR, skuID, *c.logger, convert, opts)
	if err != nil {
		return nil, err
	}
	return &sku, nil
}

func (c *NetworkController) ListInternetGateways(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListInternetGatewaysParams) (*secapi.Iterator[sdkschema.InternetGateway], error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) DeleteInternetGateway(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteInternetGatewayParams) error {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) GetInternetGateway(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.InternetGateway, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) CreateOrUpdateInternetGateway(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateInternetGatewayParams, req sdkschema.InternetGateway) (*sdkschema.InternetGateway, bool, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) ListNetworks(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListNetworksParams) (*secapi.Iterator[sdkschema.Network], error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) DeleteNetwork(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteNetworkParams) error {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) GetNetwork(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.Network, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) CreateOrUpdateNetwork(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateNetworkParams, req sdkschema.Network) (*sdkschema.Network, bool, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) ListRouteTables(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListRouteTablesParams) (*secapi.Iterator[sdkschema.RouteTable], error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) DeleteRouteTable(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteRouteTableParams) error {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) GetRouteTable(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.RouteTable, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) CreateOrUpdateRouteTable(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateRouteTableParams, req sdkschema.RouteTable) (*sdkschema.RouteTable, bool, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) ListSubnets(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListSubnetsParams) (*secapi.Iterator[sdkschema.Subnet], error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) DeleteSubnet(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteSubnetParams) error {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) GetSubnet(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.Subnet, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) CreateOrUpdateSubnet(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateSubnetParams, req sdkschema.Subnet) (*sdkschema.Subnet, bool, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) ListNics(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListNicsParams) (*secapi.Iterator[sdkschema.Nic], error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) DeleteNic(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteNicParams) error {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) GetNic(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.Nic, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) CreateOrUpdateNic(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateNicParams, req sdkschema.Nic) (*sdkschema.Nic, bool, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) ListPublicIps(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListPublicIpsParams) (*secapi.Iterator[sdkschema.PublicIp], error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) GetPublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string) (sdkschema.PublicIp, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) CreateOrUpdatePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params sdknetwork.CreateOrUpdatePublicIpParams, req sdkschema.PublicIp) (*sdkschema.PublicIp, bool, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) DeletePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params sdknetwork.DeletePublicIpParams) error {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) ListSecurityGroups(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListSecurityGroupsParams) (*secapi.Iterator[sdkschema.SecurityGroup], error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) DeleteSecurityGroup(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.DeleteSecurityGroupParams) error {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) GetSecurityGroup(ctx context.Context, tenantID, workspaceID, name string) (sdkschema.SecurityGroup, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

func (c *NetworkController) CreateOrUpdateSecurityGroup(ctx context.Context, tenantID, workspaceID, name string, params sdknetwork.CreateOrUpdateSecurityGroupParams, req sdkschema.SecurityGroup) (*sdkschema.SecurityGroup, bool, error) {
	// TODO implement me
	c.logger.Debug("implement me")
	panic("implement me")
}

// --- Helpers ---

func fromCRToSDKNetworkSKU(crNetworkSKU skuv1.NetworkSKU) sdkschema.NetworkSku {
	sdkNetworkSKU := sdkschema.NetworkSku{
		Spec: &sdkschema.NetworkSkuSpec{
			Bandwidth: crNetworkSKU.Spec.Bandwidth,
			Packets:   crNetworkSKU.Spec.Packets,
		},
		Metadata: &sdkschema.SkuResourceMetadata{
			Name: crNetworkSKU.GetName(),
		},
	}
	return sdkNetworkSKU
}
