package converter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
)

func secaPublicIp(spec publicipdom.PublicIpSpec) *publicipdom.PublicIp {
	return &publicipdom.PublicIp{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "my-public-ip"},
			Scope: res.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
			Region: "ITBG-Bergamo",
		},
		Spec: spec,
	}
}

func TestPublicIpElasticIpConverter_FromSECAToAruba(t *testing.T) {
	wantNamespace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: "test-tenant", Workspace: "test-workspace"})
	wantProjectNamespace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: "test-tenant"})

	t.Run("happy path", func(t *testing.T) {
		eip, err := converter.NewPublicIpElasticIpConverter().FromSECAToAruba(
			secaPublicIp(publicipdom.PublicIpSpec{Version: commondomain.IPVersionIPv4}))
		require.NoError(t, err)

		require.Equal(t, "my-public-ip", eip.Name)
		require.Equal(t, wantNamespace, eip.Namespace)
		require.Equal(t, "test-tenant", eip.Spec.Tenant)
		require.Equal(t, "ITBG-Bergamo", eip.Spec.Region)
		require.Equal(t, "Hour", eip.Spec.BillingPeriod)
		require.Equal(t, "test-workspace", eip.Spec.ProjectReference.Name)
		require.Equal(t, wantProjectNamespace, eip.Spec.ProjectReference.Namespace)
	})

	// Aruba always allocates the address itself, so a spec asking for a specific one cannot be
	// honoured and must not be silently ignored.
	t.Run("BYOIP is rejected", func(t *testing.T) {
		_, err := converter.NewPublicIpElasticIpConverter().FromSECAToAruba(
			secaPublicIp(publicipdom.PublicIpSpec{Address: "203.0.113.7", Version: commondomain.IPVersionIPv4}))
		require.ErrorContains(t, err, "bring-your-own-IP")
	})

	t.Run("IPv6 is rejected", func(t *testing.T) {
		_, err := converter.NewPublicIpElasticIpConverter().FromSECAToAruba(
			secaPublicIp(publicipdom.PublicIpSpec{Version: commondomain.IPVersionIPv6}))
		require.ErrorContains(t, err, "IPv6")
	})
}

func TestPublicIpElasticIpConverter_FromArubaToSECA(t *testing.T) {
	c := converter.NewPublicIpElasticIpConverter()

	eip, err := c.FromSECAToAruba(secaPublicIp(publicipdom.PublicIpSpec{Version: commondomain.IPVersionIPv4}))
	require.NoError(t, err)

	domain, err := c.FromArubaToSECA(eip)
	require.NoError(t, err)

	require.Equal(t, "my-public-ip", domain.Name)
	require.Equal(t, "test-tenant", domain.GetTenant())
	require.Equal(t, "test-workspace", domain.GetWorkspace())
	require.Equal(t, commondomain.IPVersionIPv4, domain.Spec.Version)
}
