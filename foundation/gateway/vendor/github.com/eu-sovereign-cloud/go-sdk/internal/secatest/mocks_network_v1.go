package secatest

import (
	"net/http"

	mocknetwork "github.com/eu-sovereign-cloud/go-sdk/mock/spec/foundation.network.v1"
	network "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/stretchr/testify/mock"
)

// Network Sku
func MockListNetworkSkusV1(sim *mocknetwork.MockServerInterface, resp NetworkSkuResponseV1) {
	sim.EXPECT().ListSkus(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, params network.ListSkusParams) {
			if err := configGetHttpResponse(w, networkSkusResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetNetworkSkuV1(sim *mocknetwork.MockServerInterface, resp NetworkSkuResponseV1) {
	sim.EXPECT().GetSku(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, name string) {
			if err := configGetHttpResponse(w, networkSkuResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

// Network
func MockListNetworksV1(sim *mocknetwork.MockServerInterface, resp NetworkResponseV1) {
	sim.EXPECT().ListNetworks(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, params network.ListNetworksParams) {
			if err := configGetHttpResponse(w, networksResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetNetworkV1(sim *mocknetwork.MockServerInterface, resp NetworkResponseV1) {
	sim.EXPECT().GetNetwork(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string) {
			if err := configGetHttpResponse(w, networkResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockCreateOrUpdateNetworkV1(sim *mocknetwork.MockServerInterface, resp NetworkResponseV1) {
	sim.EXPECT().CreateOrUpdateNetwork(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string, params network.CreateOrUpdateNetworkParams) {
			if err := configPutHttpResponse(w, networkResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockDeleteNetworkV1(sim *mocknetwork.MockServerInterface) {
	sim.EXPECT().DeleteNetwork(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string, params network.DeleteNetworkParams) {
			configDeleteHttpResponse(w)
		})
}

// Subnet
func MockListSubnetsV1(sim *mocknetwork.MockServerInterface, resp SubnetResponseV1) {
	sim.EXPECT().ListSubnets(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, network schema.NetworkPathParam, params network.ListSubnetsParams) {
			if err := configGetHttpResponse(w, subnetsResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetSubnetV1(sim *mocknetwork.MockServerInterface, resp SubnetResponseV1) {
	sim.EXPECT().GetSubnet(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, network schema.NetworkPathParam, params schema.ResourcePathParam) {
			if err := configGetHttpResponse(w, subnetResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockCreateOrUpdateSubnetV1(sim *mocknetwork.MockServerInterface, resp SubnetResponseV1) {
	sim.EXPECT().CreateOrUpdateSubnet(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, network schema.NetworkPathParam, name schema.ResourcePathParam, params network.CreateOrUpdateSubnetParams) {
			if err := configPutHttpResponse(w, subnetResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockDeleteSubnetV1(sim *mocknetwork.MockServerInterface) {
	sim.EXPECT().DeleteSubnet(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, network schema.NetworkPathParam, name schema.ResourcePathParam, params network.DeleteSubnetParams) {
			configDeleteHttpResponse(w)
		})
}

// Route Table
func MockListRouteTablesV1(sim *mocknetwork.MockServerInterface, resp RouteTableResponseV1) {
	sim.EXPECT().ListRouteTables(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, network schema.NetworkPathParam, params network.ListRouteTablesParams) {
			if err := configGetHttpResponse(w, routeTablesResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetRouteTableV1(sim *mocknetwork.MockServerInterface, resp RouteTableResponseV1) {
	sim.EXPECT().GetRouteTable(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, network schema.NetworkPathParam, name schema.ResourcePathParam) {
			if err := configGetHttpResponse(w, routeTableResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockCreateOrUpdateRouteTableV1(sim *mocknetwork.MockServerInterface, resp RouteTableResponseV1) {
	sim.EXPECT().CreateOrUpdateRouteTable(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, network schema.NetworkPathParam, name schema.ResourcePathParam, params network.CreateOrUpdateRouteTableParams) {
			if err := configPutHttpResponse(w, routeTableResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockDeleteRouteTableV1(sim *mocknetwork.MockServerInterface) {
	sim.EXPECT().DeleteRouteTable(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, network schema.NetworkPathParam, name schema.ResourcePathParam, params network.DeleteRouteTableParams) {
			configDeleteHttpResponse(w)
		})
}

// Internet Gateway
func MockListInternetGatewaysV1(sim *mocknetwork.MockServerInterface, resp InternetGatewayResponseV1) {
	sim.EXPECT().ListInternetGateways(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, params network.ListInternetGatewaysParams) {
			if err := configGetHttpResponse(w, internetGatewaysResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetInternetGatewayV1(sim *mocknetwork.MockServerInterface, resp InternetGatewayResponseV1) {
	sim.EXPECT().GetInternetGateway(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string) {
			if err := configGetHttpResponse(w, internetGatewayResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockCreateOrUpdateInternetGatewayV1(sim *mocknetwork.MockServerInterface, resp InternetGatewayResponseV1) {
	sim.EXPECT().CreateOrUpdateInternetGateway(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string, params network.CreateOrUpdateInternetGatewayParams) {
			if err := configPutHttpResponse(w, internetGatewayResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockDeleteInternetGatewayV1(sim *mocknetwork.MockServerInterface) {
	sim.EXPECT().DeleteInternetGateway(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string, params network.DeleteInternetGatewayParams) {
			configDeleteHttpResponse(w)
		})
}

// Security Group
func MockListSecurityGroupsV1(sim *mocknetwork.MockServerInterface, resp SecurityGroupResponseV1) {
	sim.EXPECT().ListSecurityGroups(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, params network.ListSecurityGroupsParams) {
			if err := configGetHttpResponse(w, securityGroupsResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetSecurityGroupV1(sim *mocknetwork.MockServerInterface, resp SecurityGroupResponseV1) {
	sim.EXPECT().GetSecurityGroup(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string) {
			if err := configGetHttpResponse(w, securityGroupResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockCreateOrUpdateSecurityGroupV1(sim *mocknetwork.MockServerInterface, resp SecurityGroupResponseV1) {
	sim.EXPECT().CreateOrUpdateSecurityGroup(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string, params network.CreateOrUpdateSecurityGroupParams) {
			if err := configPutHttpResponse(w, securityGroupResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockDeleteSecurityGroupV1(sim *mocknetwork.MockServerInterface) {
	sim.EXPECT().DeleteSecurityGroup(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string, params network.DeleteSecurityGroupParams) {
			configDeleteHttpResponse(w)
		})
}

// Nic
func MockListNicsV1(sim *mocknetwork.MockServerInterface, resp NicResponseV1) {
	sim.EXPECT().ListNics(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, params network.ListNicsParams) {
			if err := configGetHttpResponse(w, nicsResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetNicV1(sim *mocknetwork.MockServerInterface, resp NicResponseV1) {
	sim.EXPECT().GetNic(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string) {
			if err := configGetHttpResponse(w, nicResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockCreateOrUpdateNicV1(sim *mocknetwork.MockServerInterface, resp NicResponseV1) {
	sim.EXPECT().CreateOrUpdateNic(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string, params network.CreateOrUpdateNicParams) {
			if err := configPutHttpResponse(w, nicResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockDeleteNicV1(sim *mocknetwork.MockServerInterface) {
	sim.EXPECT().DeleteNic(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string, params network.DeleteNicParams) {
			configDeleteHttpResponse(w)
		})
}

// Public Ip
func MockListPublicIpsV1(sim *mocknetwork.MockServerInterface, resp PublicIpResponseV1) {
	sim.EXPECT().ListPublicIps(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, params network.ListPublicIpsParams) {
			if err := configGetHttpResponse(w, publicIpsResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetPublicIpV1(sim *mocknetwork.MockServerInterface, resp PublicIpResponseV1) {
	sim.EXPECT().GetPublicIp(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace string, name string) {
			if err := configGetHttpResponse(w, publicIpResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockCreateOrUpdatePublicIpV1(sim *mocknetwork.MockServerInterface, resp PublicIpResponseV1) {
	sim.EXPECT().CreateOrUpdatePublicIp(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace, name string, params network.CreateOrUpdatePublicIpParams) {
			if err := configPutHttpResponse(w, publicIpResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockDeletePublicIpV1(sim *mocknetwork.MockServerInterface) {
	sim.EXPECT().DeletePublicIp(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant string, workspace, name string, params network.DeletePublicIpParams) {
			configDeleteHttpResponse(w)
		})
}
