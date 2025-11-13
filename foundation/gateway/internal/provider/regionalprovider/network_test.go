package regionalprovider

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generatedv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/network/skus/v1"
	network "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

type NetworkTestSuite struct {
	Controller NetworkProvider
}

func NewNetworkTestSuite(t *testing.T) *NetworkTestSuite {
	t.Helper()

	// TODO: have a network controller properly initialized
	//
	// To achieve that, we need also to refactor `block-storage_test.go` on order to
	// extract the code which initializes the Kubernetes Client to a common function.
	//
	//controller, err := NewNetworkController(slog.Default(), nil)
	//require.NoError(t, err)

	return &NetworkTestSuite{
		Controller: &NetworkController{logger: slog.Default()},
	}
}

func TestNetworkController_ListSKUs(t *testing.T) {
	t.Run("should panic because the controller is not properly configured", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.Panics(t, func() {
			suite.Controller.ListSKUs(context.Background(), "", network.ListSkusParams{})
		})
	})
}

func TestNetworkController_GetSKU(t *testing.T) {
	t.Run("should panic because the controller is not properly configured", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.Panics(t, func() {
			suite.Controller.GetSKU(context.Background(), "", "")
		})
	})
}

func TestNetworkController_ListInternetGateways(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.ListInternetGateways(context.Background(), "", "", network.ListInternetGatewaysParams{})
		})
	})
}

func TestNetworkController_DeleteInternetGateway(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.DeleteInternetGateway(context.Background(), "", "", "", network.DeleteInternetGatewayParams{})
		})
	})
}

func TestNetworkController_GetInternetGateway(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.GetInternetGateway(context.Background(), "", "", "")
		})
	})
}

func TestNetworkController_CreateOrUpdateInternetGateway(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.CreateOrUpdateInternetGateway(context.Background(), "", "", "", network.CreateOrUpdateInternetGatewayParams{}, sdkschema.InternetGateway{})
		})
	})
}

func TestNetworkController_ListNetworks(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.ListNetworks(context.Background(), "", "", network.ListNetworksParams{})
		})
	})
}

func TestNetworkController_DeleteNetwork(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.DeleteNetwork(context.Background(), "", "", "", network.DeleteNetworkParams{})
		})
	})
}

func TestNetworkController_GetNetwork(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.GetNetwork(context.Background(), "", "", "")
		})
	})
}

func TestNetworkController_CreateOrUpdateNetwork(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.CreateOrUpdateNetwork(context.Background(), "", "", "", network.CreateOrUpdateNetworkParams{}, sdkschema.Network{})
		})
	})
}

func TestNetworkController_ListRouteTables(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.ListRouteTables(context.Background(), "", "", network.ListRouteTablesParams{})
		})
	})
}

func TestNetworkController_DeleteRouteTable(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.DeleteRouteTable(context.Background(), "", "", "", network.DeleteRouteTableParams{})
		})
	})
}

func TestNetworkController_GetRouteTable(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.GetRouteTable(context.Background(), "", "", "")
		})
	})
}

func TestNetworkController_CreateOrUpdateRouteTable(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.CreateOrUpdateRouteTable(context.Background(), "", "", "", network.CreateOrUpdateRouteTableParams{}, sdkschema.RouteTable{})
		})
	})
}

func TestNetworkController_ListSubnets(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.ListSubnets(context.Background(), "", "", network.ListSubnetsParams{})
		})
	})
}

func TestNetworkController_DeleteSubnet(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.DeleteSubnet(context.Background(), "", "", "", network.DeleteSubnetParams{})
		})
	})
}

func TestNetworkController_GetSubnet(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.GetSubnet(context.Background(), "", "", "")
		})
	})
}

func TestNetworkController_CreateOrUpdateSubnet(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.CreateOrUpdateSubnet(context.Background(), "", "", "", network.CreateOrUpdateSubnetParams{}, sdkschema.Subnet{})
		})
	})
}

func TestNetworkController_ListNics(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.ListNics(context.Background(), "", "", network.ListNicsParams{})
		})
	})
}

func TestNetworkController_DeleteNic(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.DeleteNic(context.Background(), "", "", "", network.DeleteNicParams{})
		})
	})
}

func TestNetworkController_GetNic(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.GetNic(context.Background(), "", "", "")
		})
	})
}

func TestNetworkController_CreateOrUpdateNic(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.CreateOrUpdateNic(context.Background(), "", "", "", network.CreateOrUpdateNicParams{}, sdkschema.Nic{})
		})
	})
}

func TestNetworkController_ListPublicIps(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.ListPublicIps(context.Background(), "", "", network.ListPublicIpsParams{})
		})
	})
}

func TestNetworkController_GetPublicIp(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.GetPublicIp(context.Background(), "", "", "")
		})
	})
}

func TestNetworkController_CreateOrUpdatePublicIp(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.CreateOrUpdatePublicIp(context.Background(), "", "", "", network.CreateOrUpdatePublicIpParams{}, sdkschema.PublicIp{})
		})
	})
}

func TestNetworkController_DeletePublicIp(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.DeletePublicIp(context.Background(), "", "", "", network.DeletePublicIpParams{})
		})
	})
}

func TestNetworkController_ListSecurityGroups(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.ListSecurityGroups(context.Background(), "", "", network.ListSecurityGroupsParams{})
		})
	})
}

func TestNetworkController_GetSecurityGroup(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.GetSecurityGroup(context.Background(), "", "", "")
		})
	})
}

func TestNetworkController_CreateOrUpdateSecurityGroup(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.CreateOrUpdateSecurityGroup(context.Background(), "", "", "", network.CreateOrUpdateSecurityGroupParams{}, sdkschema.SecurityGroup{})
		})
	})
}

func TestNetworkController_DeleteSecurityGroup(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewNetworkTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Controller.DeleteSecurityGroup(context.Background(), "", "", "", network.DeleteSecurityGroupParams{})
		})
	})
}

// --- Helpers ---

// newNetworkSKUCR constructs a typed NetworkSKU CR.
func newNetworkSKUCR(name, tenant string, labels map[string]string, bandwidth, packets int, setVersionAndTimestamp bool) *skuv1.NetworkSKU {
	if labels == nil {
		labels = map[string]string{}
	}
	cr := &skuv1.NetworkSKU{
		TypeMeta:   metav1.TypeMeta{Kind: "NetworkSKU", APIVersion: fmt.Sprintf("%s/%s", skuv1.NetworkSKUGVR.Group, skuv1.NetworkSKUGVR.Version)},
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels, Namespace: tenant},
		Spec:       generatedv1.NetworkSkuSpec{Bandwidth: bandwidth, Packets: packets},
	}
	if setVersionAndTimestamp {
		cr.SetCreationTimestamp(metav1.Time{Time: time.Unix(1700000000, 0)})
		cr.SetResourceVersion("1")
	}
	return cr
}
