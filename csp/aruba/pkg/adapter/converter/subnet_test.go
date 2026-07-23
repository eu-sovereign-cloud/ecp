package converter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
)

func secaSubnet(cidr subnetdom.CIDR) *subnetdom.Subnet {
	return &subnetdom.Subnet{
		RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: "my-subnet"},
				Scope: res.Scope{
					Tenant:    "test-tenant",
					Workspace: "test-workspace",
				},
				Region: "ITBG-Bergamo",
			},
			Network: "my-network",
		},
		Spec: subnetdom.SubnetSpec{
			Cidr: cidr,
			Zone: "ITBG-1",
		},
	}
}

func TestSubnetConverter_FromSECAToAruba(t *testing.T) {
	// The subnet lands in its network's own namespace, but the VPC it references lives one
	// level up, in the workspace namespace.
	wantNamespace := k8sadapter.ComputeNetworkNamespace(secaSubnet(subnetdom.CIDR{IPv4: "10.0.1.0/24"}))
	wantVPCNamespace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: "test-tenant", Workspace: "test-workspace"})
	wantProjectNamespace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: "test-tenant"})

	t.Run("happy path", func(t *testing.T) {
		subnet, err := converter.NewSubnetConverter().FromSECAToAruba(secaSubnet(subnetdom.CIDR{IPv4: "10.0.1.0/24"}))
		require.NoError(t, err)

		require.Equal(t, "my-subnet", subnet.Name)
		require.Equal(t, wantNamespace, subnet.Namespace)
		require.NotEqual(t, wantVPCNamespace, subnet.Namespace, "network namespace must differ from workspace namespace")
		require.Equal(t, "10.0.1.0/24", subnet.Spec.CIDR)
		require.Equal(t, "Advanced", subnet.Spec.Type)
		require.True(t, subnet.Spec.DHCP.Enabled)
		require.Equal(t, "my-network", subnet.Spec.VPCReference.Name)
		require.Equal(t, wantVPCNamespace, subnet.Spec.VPCReference.Namespace)
		require.Equal(t, "test-workspace", subnet.Spec.ProjectReference.Name)
		require.Equal(t, wantProjectNamespace, subnet.Spec.ProjectReference.Namespace)
	})

	t.Run("dual stack keeps the IPv4 range", func(t *testing.T) {
		subnet, err := converter.NewSubnetConverter().FromSECAToAruba(
			secaSubnet(subnetdom.CIDR{IPv4: "10.0.1.0/24", IPv6: "2001:db8::/64"}))
		require.NoError(t, err)
		require.Equal(t, "10.0.1.0/24", subnet.Spec.CIDR)
	})

	t.Run("IPv6 only is rejected", func(t *testing.T) {
		_, err := converter.NewSubnetConverter().FromSECAToAruba(secaSubnet(subnetdom.CIDR{IPv6: "2001:db8::/64"}))
		require.ErrorContains(t, err, "IPv4 CIDR")
	})
}

// The subnet converter resolves its VPC reference by recomputing what the network converter
// produced. Nothing at runtime cross-checks the two, so a drift between them would leave every
// subnet waiting for a VPC that exists under another identity.
func TestSubnetConverter_VPCReferenceMatchesNetworkConverter(t *testing.T) {
	vpc, err := converter.NewNetworkVPCConverter().FromSECAToAruba(secaNetwork("ITBG-Bergamo"))
	require.NoError(t, err)

	subnet, err := converter.NewSubnetConverter().FromSECAToAruba(secaSubnet(subnetdom.CIDR{IPv4: "10.0.1.0/24"}))
	require.NoError(t, err)

	require.Equal(t, vpc.Name, subnet.Spec.VPCReference.Name)
	require.Equal(t, vpc.Namespace, subnet.Spec.VPCReference.Namespace)

	// Both must also agree on the Project they hang off.
	require.Equal(t, vpc.Spec.ProjectReference, subnet.Spec.ProjectReference)
}

func TestSubnetConverter_FromArubaToSECA(t *testing.T) {
	c := converter.NewSubnetConverter()

	subnet, err := c.FromSECAToAruba(secaSubnet(subnetdom.CIDR{IPv4: "10.0.1.0/24"}))
	require.NoError(t, err)

	domain, err := c.FromArubaToSECA(subnet)
	require.NoError(t, err)

	require.Equal(t, "my-subnet", domain.Name)
	require.Equal(t, "test-tenant", domain.GetTenant())
	require.Equal(t, "test-workspace", domain.GetWorkspace())
	require.Equal(t, "my-network", domain.GetNetwork())
	require.Equal(t, "10.0.1.0/24", domain.Spec.Cidr.IPv4)
}
