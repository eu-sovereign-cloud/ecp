package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
)

func TestPublicIpConversionRoundTrip(t *testing.T) {
	// Reference.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	attachedTo := commondomain.Reference{Resource: "nics/nic1"}
	in := &publicipdom.PublicIp{
		Spec: publicipdom.PublicIpSpec{
			Address: "203.0.113.5",
			Version: commondomain.IPVersionIPv4,
		},
	}
	in.Name = "ip1"
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Provider = publicipdom.ProviderID
	in.Region = "r1"
	in.Status = &publicipdom.PublicIpStatus{
		Status:     commondomain.Status{State: commondomain.ResourceStateActive},
		AttachedTo: &attachedTo,
		IpAddress:  "203.0.113.5",
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := PublicIpToCR(in)
	require.NoError(t, err)

	out, err := PublicIpFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Region, out.Region)
	require.Equal(t, in.Spec.Address, out.Spec.Address)
	require.Equal(t, in.Spec.Version, out.Spec.Version)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
	require.Equal(t, in.Status.IpAddress, out.Status.IpAddress)
	require.NotNil(t, out.Status.AttachedTo)
	require.Equal(t, *in.Status.AttachedTo, *out.Status.AttachedTo)
}

func TestPublicIpToCR_NilAttachedTo(t *testing.T) {
	in := &publicipdom.PublicIp{
		Spec: publicipdom.PublicIpSpec{
			Version: commondomain.IPVersionIPv6,
		},
	}
	in.Name = "ip1"

	cr, err := PublicIpToCR(in)
	require.NoError(t, err)

	out, err := PublicIpFromCR(cr)
	require.NoError(t, err)
	require.Nil(t, out.Status.AttachedTo)
	require.Equal(t, commondomain.IPVersionIPv6, out.Spec.Version)
}
