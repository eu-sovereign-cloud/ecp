package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
)

func TestInternetGatewayConversionRoundTrip(t *testing.T) {
	in := &internetgatewaydom.InternetGateway{
		Spec: internetgatewaydom.InternetGatewaySpec{
			EgressOnly: true,
		},
	}
	in.Name = "ig1"
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Provider = internetgatewaydom.ProviderID
	in.Region = "r1"
	in.Status = &internetgatewaydom.InternetGatewayStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := InternetGatewayToCR(in)
	require.NoError(t, err)

	out, err := InternetGatewayFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Region, out.Region)
	require.Equal(t, in.Spec.EgressOnly, out.Spec.EgressOnly)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
}

func TestInternetGatewayToCR_EgressOnlyFalse(t *testing.T) {
	in := &internetgatewaydom.InternetGateway{
		Spec: internetgatewaydom.InternetGatewaySpec{
			EgressOnly: false,
		},
	}
	in.Name = "ig1"

	cr, err := InternetGatewayToCR(in)
	require.NoError(t, err)

	out, err := InternetGatewayFromCR(cr)
	require.NoError(t, err)
	require.False(t, out.Spec.EgressOnly)
	require.Equal(t, commondomain.ResourceStatePending, out.Status.Conditions[0].State)
}
