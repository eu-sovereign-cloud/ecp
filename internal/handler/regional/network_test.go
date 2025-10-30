package regionalhandler

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
)

type NetworkTestSuite struct {
	Handler *Network
}

func NewNetworkTestSuite(t *testing.T) *NetworkTestSuite {
	t.Helper()

	return &NetworkTestSuite{
		Handler: &Network{},
	}

}

func TestNetwoekHandler_ListSkus(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListSkus(nil, nil, "", sdknetwork.ListSkusParams{})
		})
	})
}

func TestNetwoekHandler_GetSku(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetSku(nil, nil, "", "")
		})
	})
}

func TestNetwoekHandler_ListInternetGateways(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListInternetGateways(nil, nil, "", "", sdknetwork.ListInternetGatewaysParams{})
		})
	})
}

func TestNetwoekHandler_DeleteInternetGateway(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteInternetGateway(nil, nil, "", "", "", sdknetwork.DeleteInternetGatewayParams{})
		})
	})
}

func TestNetwoekHandler_GetInternetGateway(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetInternetGateway(nil, nil, "", "", "")
		})
	})
}

func TestNetwoekHandler_CreateOrUpdateInternetGateway(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateInternetGateway(nil, nil, "", "", "", sdknetwork.CreateOrUpdateInternetGatewayParams{})
		})
	})
}

func TestNetwoekHandler_ListNetworks(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListNetworks(nil, nil, "", "", sdknetwork.ListNetworksParams{})
		})
	})
}

func TestNetwoekHandler_DeleteNetwork(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteNetwork(nil, nil, "", "", "", sdknetwork.DeleteNetworkParams{})
		})
	})
}

func TestNetwoekHandler_GetNetwork(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetNetwork(nil, nil, "", "", "")
		})
	})
}

func TestNetwoekHandler_CreateOrUpdateNetwork(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateNetwork(nil, nil, "", "", "", sdknetwork.CreateOrUpdateNetworkParams{})
		})
	})
}

func TestNetwoekHandler_ListRouteTables(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListRouteTables(nil, nil, "", "", "", sdknetwork.ListRouteTablesParams{})
		})
	})
}

func TestNetwoekHandler_DeleteRouteTable(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteRouteTable(nil, nil, "", "", "", "", sdknetwork.DeleteRouteTableParams{})
		})
	})
}

func TestNetwoekHandler_GetRouteTable(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetRouteTable(nil, nil, "", "", "", "")
		})
	})
}

func TestNetwoekHandler_CreateOrUpdateRouteTable(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateRouteTable(nil, nil, "", "", "", "", sdknetwork.CreateOrUpdateRouteTableParams{})
		})
	})
}

func TestNetwoekHandler_ListSubnets(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListSubnets(nil, nil, "", "", "", sdknetwork.ListSubnetsParams{})
		})
	})
}

func TestNetwoekHandler_DeleteSubnet(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteSubnet(nil, nil, "", "", "", "", sdknetwork.DeleteSubnetParams{})
		})
	})
}

func TestNetwoekHandler_GetSubnet(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetSubnet(nil, nil, "", "", "", "")
		})
	})
}

func TestNetwoekHandler_CreateOrUpdateSubnet(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateSubnet(nil, nil, "", "", "", "", sdknetwork.CreateOrUpdateSubnetParams{})
		})
	})
}

func TestNetwoekHandler_ListNics(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListNics(nil, nil, "", "", sdknetwork.ListNicsParams{})
		})
	})
}

func TestNetwoekHandler_DeleteNic(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteNic(nil, nil, "", "", "", sdknetwork.DeleteNicParams{})
		})
	})
}

func TestNetwoekHandler_GetNic(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetNic(nil, nil, "", "", "")
		})
	})
}

func TestNetwoekHandler_CreateOrUpdateNic(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateNic(nil, nil, "", "", "", sdknetwork.CreateOrUpdateNicParams{})
		})
	})
}

func TestNetwoekHandler_ListPublicIps(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListPublicIps(nil, nil, "", "", sdknetwork.ListPublicIpsParams{})
		})
	})
}

func TestNetwoekHandler_DeletePublicIp(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeletePublicIp(nil, nil, "", "", "", sdknetwork.DeletePublicIpParams{})
		})
	})
}

func TestNetwoekHandler_GetPublicIp(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetPublicIp(nil, nil, "", "", "")
		})
	})
}

func TestNetwoekHandler_CreateOrUpdatePublicIp(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdatePublicIp(nil, nil, "", "", "", sdknetwork.CreateOrUpdatePublicIpParams{})
		})
	})
}

func TestNetwoekHandler_ListSecurityGroups(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListSecurityGroups(nil, nil, "", "", sdknetwork.ListSecurityGroupsParams{})
		})
	})
}

func TestNetwoekHandler_DeleteSecurityGroup(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteSecurityGroup(nil, nil, "", "", "", sdknetwork.DeleteSecurityGroupParams{})
		})
	})
}

func TestNetwoekHandler_GetSecurityGroup(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetSecurityGroup(nil, nil, "", "", "")
		})
	})
}

func TestNetwoekHandler_CreateOrUpdateSecurityGroup(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateSecurityGroup(nil, nil, "", "", "", sdknetwork.CreateOrUpdateSecurityGroupParams{})
		})
	})
}
