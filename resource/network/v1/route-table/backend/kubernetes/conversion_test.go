package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
)

func TestRouteTableConversionRoundTrip(t *testing.T) {
	in := &routetabledom.RouteTable{
		Spec: routetabledom.RouteTableSpec{
			Routes: []routetabledom.RouteSpec{
				{
					DestinationCidrBlock: "10.0.0.0/24",
					// TODO_TEST_238_239
					// TargetRef:            commondomain.Reference{Resource: "instances/inst1"},
					TargetRef: commondomain.Reference{Resource: "instances/inst1"},
				},
			},
		},
	}
	in.Name = "rt1"
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Network = "n1"
	in.Provider = routetabledom.ProviderID
	in.Region = "r1"
	in.Status = &routetabledom.RouteTableStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
		Routes: []routetabledom.RouteStatus{
			{State: commondomain.ResourceStateActive},
		},
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := RouteTableToCR(in)
	require.NoError(t, err)

	out, err := RouteTableFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Network, out.Network)
	require.Equal(t, in.Region, out.Region)
	require.Len(t, out.Spec.Routes, 1)
	require.Equal(t, "10.0.0.0/24", out.Spec.Routes[0].DestinationCidrBlock)
	require.Equal(t, "instances/inst1", out.Spec.Routes[0].TargetRef.Resource)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
	require.Len(t, out.Status.Routes, 1)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.Routes[0].State)
}

func TestRouteTableToCR_DefaultPendingCondition(t *testing.T) {
	in := &routetabledom.RouteTable{
		Spec: routetabledom.RouteTableSpec{
			Routes: []routetabledom.RouteSpec{{DestinationCidrBlock: "10.0.0.0/24"}},
		},
	}
	in.Name = "rt1"

	cr, err := RouteTableToCR(in)
	require.NoError(t, err)

	out, err := RouteTableFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, commondomain.ResourceStatePending, out.Status.Conditions[0].State)
}

func TestRouteTableToCR_UsesNetworkNamespace(t *testing.T) {
	withNetwork := &routetabledom.RouteTable{}
	withNetwork.Name = "rt1"
	withNetwork.Tenant = "t1"
	withNetwork.Workspace = "w1"
	withNetwork.Network = "n1"

	cr, err := RouteTableToCR(withNetwork)
	require.NoError(t, err)
	require.NotEmpty(t, cr.GetNamespace())

	other := &routetabledom.RouteTable{}
	other.Name = "rt1"
	other.Tenant = "t1"
	other.Workspace = "w1"
	other.Network = "n2"

	otherCR, err := RouteTableToCR(other)
	require.NoError(t, err)
	require.NotEqual(t, cr.GetNamespace(), otherCR.GetNamespace(),
		"different networks must map to different namespaces")
}
