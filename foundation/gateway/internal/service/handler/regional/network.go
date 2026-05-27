package regionalhandler

import (
	"log/slog"
	"net/http"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/regional/network"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/service/handler"
	apinetwork "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/network"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/config"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional/consts"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

type Network struct {
	ListSKUs          *network.ListSKUs
	GetSKU            *network.GetSKU
	ListNetworksCtrl  *network.ListNetworks
	GetNetworkCtrl    *network.GetNetwork
	CreateNetworkCtrl *network.CreateNetwork
	UpdateNetworkCtrl *network.UpdateNetwork
	DeleteNetworkCtrl *network.DeleteNetwork
	Logger            *slog.Logger
}

func (n Network) ListSecurityGroupRules(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListSecurityGroupRulesParams) {
	// TODO implement me
	panic("implement me")
}

func (n Network) DeleteSecurityGroupRule(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSecurityGroupRuleParams) {
	// TODO implement me
	panic("implement me")
}

func (n Network) GetSecurityGroupRule(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	panic("implement me")
}

func (n Network) CreateOrUpdateSecurityGroupRule(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSecurityGroupRuleParams) {
	// TODO implement me
	panic("implement me")
}

var _ sdknetwork.ServerInterface = (*Network)(nil) // Ensure Network implements the sdknetwork.ServerInterface.

func (n Network) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdknetwork.ListSkusParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) ListInternetGateways(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListInternetGatewaysParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) DeleteInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteInternetGatewayParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) GetInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) CreateOrUpdateInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateInternetGatewayParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) ListNetworks(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListNetworksParams) {
	handler.HandleList(w, r, n.Logger.With("provider", "network").With("resource", "network"),
		apinetwork.ListParamsFromAPI(params, tenant, workspace),
		n.ListNetworksCtrl,
		apinetwork.DomainToAPIIterator,
	)
}

func (n Network) DeleteNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteNetworkParams) {
	metadata := regional.Metadata{
		CommonMetadata: model.CommonMetadata{
			Name:     name,
			Provider: consts.NetworkProvider,
		},
		Scope: scope.Scope{
			Tenant:    tenant,
			Workspace: workspace,
		},
		Region: config.Singleton().Region(),
	}
	if params.IfUnmodifiedSince != nil {
		metadata.ResourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	handler.HandleDelete(w, r, n.Logger.With("provider", "network").With("resource", "network"),
		&metadata,
		n.DeleteNetworkCtrl,
	)
}

func (n Network) GetNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	handler.HandleGet(w, r, n.Logger.With("provider", "network").With("resource", "network"),
		&regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name:     name,
				Provider: consts.NetworkProvider,
			},
			Scope: scope.Scope{
				Tenant:    tenant,
				Workspace: workspace,
			},
			Region: config.Singleton().Region(),
		},
		n.GetNetworkCtrl,
		apinetwork.DomainToAPIWithVerb(http.MethodGet),
	)
}

func (n Network) CreateOrUpdateNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateNetworkParams) {
	var resourceVersion string
	if params.IfUnmodifiedSince != nil {
		resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}

	handler.HandleUpsert(w, r, n.Logger.With("provider", "network").With("resource", "network"),
		handler.UpsertOptions[sdkschema.Network, *regional.NetworkDomain, *sdkschema.Network]{
			Params: &regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					Name:            name,
					Provider:        consts.NetworkProvider,
					ResourceVersion: resourceVersion,
				},
				Scope: scope.Scope{
					Tenant:    tenant,
					Workspace: workspace,
				},
				Region: config.Singleton().Region(),
			},
			Creator:     n.CreateNetworkCtrl,
			Updater:     n.UpdateNetworkCtrl,
			APIToDomain: apinetwork.APIToNetworkDomain,
			DomainToAPI: apinetwork.DomainToAPIWithVerb(http.MethodPut),
		},
	)
}

func (n Network) ListRouteTables(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, params sdknetwork.ListRouteTablesParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) DeleteRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteRouteTableParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) GetRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) CreateOrUpdateRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateRouteTableParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) ListSubnets(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, params sdknetwork.ListSubnetsParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) DeleteSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSubnetParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) GetSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) CreateOrUpdateSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSubnetParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) ListNics(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListNicsParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) DeleteNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteNicParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) GetNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) CreateOrUpdateNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateNicParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) ListPublicIps(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListPublicIpsParams) {
	// TODO implement me
	// n.Controller.ListPublicIps()
	n.Logger.Debug("implement me")
}

func (n Network) DeletePublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeletePublicIpParams) {
	// TODO implement me
	// n.Controller.DeletePublicIp()
	n.Logger.Debug("implement me")
}

func (n Network) GetPublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	// n.Controller.GetPublicIp()
	n.Logger.Debug("implement me")
}

func (n Network) CreateOrUpdatePublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdatePublicIpParams) {
	// TODO implement me
	// n.Controller.CreateOrUpdatePublicIp()
	n.Logger.Debug("implement me")
}

func (n Network) ListSecurityGroups(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListSecurityGroupsParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) DeleteSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSecurityGroupParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) GetSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	// TODO implement me
	n.Logger.Debug("implement me")
}

func (n Network) CreateOrUpdateSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSecurityGroupParams) {
	// TODO implement me
	n.Logger.Debug("implement me")
}
