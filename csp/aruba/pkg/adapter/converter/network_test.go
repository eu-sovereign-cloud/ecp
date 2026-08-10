package converter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
)

func secaNetwork(region string) *netdom.Network {
	return &netdom.Network{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "my-network"},
			Scope: res.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
			Region: region,
		},
		Spec: netdom.NetworkSpec{
			CIDR: netdom.CIDR{IPv4: "10.0.0.0/16"},
		},
	}
}

func TestNetworkVPCConverter_FromSECAToAruba(t *testing.T) {
	// The VPC lands in the workspace namespace and points at the Project named after the
	// workspace, which lives in the tenant namespace.
	wantNamespace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: "test-tenant", Workspace: "test-workspace"})
	wantProjectNamespace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: "test-tenant"})

	t.Run("happy path", func(t *testing.T) {
		vpc, err := converter.NewNetworkVPCConverter().FromSECAToAruba(secaNetwork("ITBG-Bergamo"))
		require.NoError(t, err)

		require.Equal(t, "my-network", vpc.Name)
		require.Equal(t, wantNamespace, vpc.Namespace)
		require.Equal(t, "test-tenant", vpc.Spec.Tenant)
		require.Equal(t, "ITBG-Bergamo", vpc.Spec.Region)
		require.Equal(t, "test-workspace", vpc.Spec.ProjectReference.Name)
		require.Equal(t, wantProjectNamespace, vpc.Spec.ProjectReference.Namespace)
		require.Equal(t, "test-workspace", vpc.Labels["seca.network/workspace"])
		require.Equal(t, "test-tenant", vpc.Labels["seca.network/tenant"])
	})

	t.Run("region defaults when unset", func(t *testing.T) {
		vpc, err := converter.NewNetworkVPCConverter().FromSECAToAruba(secaNetwork(""))
		require.NoError(t, err)
		require.Equal(t, "ITBG-Bergamo", vpc.Spec.Region)
	})
}

func TestNetworkVPCConverter_FromArubaToSECA(t *testing.T) {
	c := converter.NewNetworkVPCConverter()

	t.Run("happy path", func(t *testing.T) {
		vpc, err := c.FromSECAToAruba(secaNetwork("ITBG-Bergamo"))
		require.NoError(t, err)

		network, err := c.FromArubaToSECA(vpc)
		require.NoError(t, err)

		require.Equal(t, "my-network", network.Name)
		require.Equal(t, "test-tenant", network.GetTenant())
		require.Equal(t, "test-workspace", network.GetWorkspace())
	})

	t.Run("missing tenant", func(t *testing.T) {
		vpc, err := c.FromSECAToAruba(secaNetwork("ITBG-Bergamo"))
		require.NoError(t, err)
		vpc.Spec.Tenant = ""
		vpc.Labels = nil

		_, err = c.FromArubaToSECA(vpc)
		require.ErrorContains(t, err, "tenant is missing")
	})
}
