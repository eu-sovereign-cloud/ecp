package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
)

func TestSubnetConversionRoundTrip(t *testing.T) {
	in := &subnetdom.Subnet{
		Spec: subnetdom.SubnetSpec{
			Cidr:          subnetdom.CIDR{IPv4: "10.0.0.0/24"},
			RouteTableRef: commondomain.Reference{Resource: "route-tables/rt1"},
			SkuRef:        commondomain.Reference{Resource: "network-skus/sku1"},
			Zone:          "zone-a",
		},
	}
	in.Name = "sn1"
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Network = "n1"
	in.Provider = subnetdom.ProviderID
	in.Region = "r1"
	in.Status = &subnetdom.SubnetStatus{
		Status:        commondomain.Status{State: commondomain.ResourceStateActive},
		Cidr:          &subnetdom.CIDR{IPv4: "10.0.0.0/24"},
		RouteTableRef: &commondomain.Reference{Resource: "route-tables/rt1"},
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := SubnetToCR(in)
	require.NoError(t, err)

	out, err := SubnetFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Network, out.Network)
	require.Equal(t, in.Region, out.Region)
	require.Equal(t, "10.0.0.0/24", out.Spec.Cidr.IPv4)
	require.Equal(t, "route-tables/rt1", out.Spec.RouteTableRef.Resource)
	require.Equal(t, "network-skus/sku1", out.Spec.SkuRef.Resource)
	require.Equal(t, "zone-a", out.Spec.Zone)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
	require.NotNil(t, out.Status.Cidr)
	require.Equal(t, "10.0.0.0/24", out.Status.Cidr.IPv4)
	require.NotNil(t, out.Status.RouteTableRef)
	require.Equal(t, "route-tables/rt1", out.Status.RouteTableRef.Resource)
}

func TestSubnetToCR_DefaultPendingCondition(t *testing.T) {
	in := &subnetdom.Subnet{
		Spec: subnetdom.SubnetSpec{
			Cidr: subnetdom.CIDR{IPv4: "10.0.0.0/24"},
			Zone: "zone-a",
		},
	}
	in.Name = "sn1"

	cr, err := SubnetToCR(in)
	require.NoError(t, err)

	out, err := SubnetFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, commondomain.ResourceStatePending, out.Status.Conditions[0].State)
}

func TestSubnetToCR_UsesNetworkNamespace(t *testing.T) {
	withNetwork := &subnetdom.Subnet{}
	withNetwork.Name = "sn1"
	withNetwork.Tenant = "t1"
	withNetwork.Workspace = "w1"
	withNetwork.Network = "n1"

	cr, err := SubnetToCR(withNetwork)
	require.NoError(t, err)
	require.NotEmpty(t, cr.GetNamespace())

	other := &subnetdom.Subnet{}
	other.Name = "sn1"
	other.Tenant = "t1"
	other.Workspace = "w1"
	other.Network = "n2"

	otherCR, err := SubnetToCR(other)
	require.NoError(t, err)
	require.NotEqual(t, cr.GetNamespace(), otherCR.GetNamespace(),
		"different networks must map to different namespaces")
}

func TestSubnetConversion_SkuRefOptional(t *testing.T) {
	in := &subnetdom.Subnet{
		Spec: subnetdom.SubnetSpec{
			Cidr:          subnetdom.CIDR{IPv4: "10.0.0.0/24"},
			RouteTableRef: commondomain.Reference{Resource: "route-tables/rt1"},
			Zone:          "zone-a",
		},
	}
	in.Name = "sn1"

	cr, err := SubnetToCR(in)
	require.NoError(t, err)

	out, err := SubnetFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, commondomain.Reference{}, out.Spec.SkuRef)
}
