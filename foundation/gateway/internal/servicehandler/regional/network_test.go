package regionalhandler

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/regional"
)

type NetworkTestSuite struct {
	Handler *Network
}

func NewNetworkTestSuite(t *testing.T) *NetworkTestSuite {
	t.Helper()

	return &NetworkTestSuite{
		Handler: &Network{
			regional.NetworkController{}, slog.Default(),
		},
	}
}

func TestNetworkHandler_ListSkus(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListSkus(nil, nil, "", sdknetwork.ListSkusParams{})
		})
	})
}

func TestNetworkHandler_GetSku(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetSku(nil, nil, "", "")
		})
	})
}

func TestNetworkHandler_ListInternetGateways(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListInternetGateways(nil, nil, "", "", sdknetwork.ListInternetGatewaysParams{})
		})
	})
}

func TestNetworkHandler_DeleteInternetGateway(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteInternetGateway(nil, nil, "", "", "", sdknetwork.DeleteInternetGatewayParams{})
		})
	})
}

func TestNetworkHandler_GetInternetGateway(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetInternetGateway(nil, nil, "", "", "")
		})
	})
}

func TestNetworkHandler_CreateOrUpdateInternetGateway(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateInternetGateway(nil, nil, "", "", "", sdknetwork.CreateOrUpdateInternetGatewayParams{})
		})
	})
}

func TestNetworkHandler_ListNetworks(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListNetworks(nil, nil, "", "", sdknetwork.ListNetworksParams{})
		})
	})
}

func TestNetworkHandler_DeleteNetwork(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteNetwork(nil, nil, "", "", "", sdknetwork.DeleteNetworkParams{})
		})
	})
}

func TestNetworkHandler_GetNetwork(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetNetwork(nil, nil, "", "", "")
		})
	})
}

func TestNetworkHandler_CreateOrUpdateNetwork(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateNetwork(nil, nil, "", "", "", sdknetwork.CreateOrUpdateNetworkParams{})
		})
	})
}

func TestNetworkHandler_ListRouteTables(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListRouteTables(nil, nil, "", "", "", sdknetwork.ListRouteTablesParams{})
		})
	})
}

func TestNetworkHandler_DeleteRouteTable(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteRouteTable(nil, nil, "", "", "", "", sdknetwork.DeleteRouteTableParams{})
		})
	})
}

func TestNetworkHandler_GetRouteTable(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetRouteTable(nil, nil, "", "", "", "")
		})
	})
}

func TestNetworkHandler_CreateOrUpdateRouteTable(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateRouteTable(nil, nil, "", "", "", "", sdknetwork.CreateOrUpdateRouteTableParams{})
		})
	})
}

func TestNetworkHandler_ListSubnets(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListSubnets(nil, nil, "", "", "", sdknetwork.ListSubnetsParams{})
		})
	})
}

func TestNetworkHandler_DeleteSubnet(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteSubnet(nil, nil, "", "", "", "", sdknetwork.DeleteSubnetParams{})
		})
	})
}

func TestNetworkHandler_GetSubnet(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetSubnet(nil, nil, "", "", "", "")
		})
	})
}

func TestNetworkHandler_CreateOrUpdateSubnet(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateSubnet(nil, nil, "", "", "", "", sdknetwork.CreateOrUpdateSubnetParams{})
		})
	})
}

func TestNetworkHandler_ListNics(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListNics(nil, nil, "", "", sdknetwork.ListNicsParams{})
		})
	})
}

func TestNetworkHandler_DeleteNic(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteNic(nil, nil, "", "", "", sdknetwork.DeleteNicParams{})
		})
	})
}

func TestNetworkHandler_GetNic(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetNic(nil, nil, "", "", "")
		})
	})
}

func TestNetworkHandler_CreateOrUpdateNic(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateNic(nil, nil, "", "", "", sdknetwork.CreateOrUpdateNicParams{})
		})
	})
}

func TestNetworkHandler_ListPublicIps(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListPublicIps(nil, nil, "", "", sdknetwork.ListPublicIpsParams{})
		})
	})
}

func TestNetworkHandler_DeletePublicIp(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeletePublicIp(nil, nil, "", "", "", sdknetwork.DeletePublicIpParams{})
		})
	})
}

func TestNetworkHandler_GetPublicIp(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetPublicIp(nil, nil, "", "", "")
		})
	})
}

func TestNetworkHandler_CreateOrUpdatePublicIp(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdatePublicIp(nil, nil, "", "", "", sdknetwork.CreateOrUpdatePublicIpParams{})
		})
	})
}

func TestNetworkHandler_ListSecurityGroups(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListSecurityGroups(nil, nil, "", "", sdknetwork.ListSecurityGroupsParams{})
		})
	})
}

func TestNetworkHandler_DeleteSecurityGroup(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteSecurityGroup(nil, nil, "", "", "", sdknetwork.DeleteSecurityGroupParams{})
		})
	})
}

func TestNetworkHandler_GetSecurityGroup(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetSecurityGroup(nil, nil, "", "", "")
		})
	})
}

func TestNetworkHandler_CreateOrUpdateSecurityGroup(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateSecurityGroup(nil, nil, "", "", "", sdknetwork.CreateOrUpdateSecurityGroupParams{})
		})
	})
}
