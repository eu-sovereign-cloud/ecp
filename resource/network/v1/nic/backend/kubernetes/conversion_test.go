package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
)

func TestNicConversionRoundTrip(t *testing.T) {
	in := &nicdom.Nic{
		Spec: nicdom.NicSpec{
			Addresses:         []string{"10.0.0.5"},
			SubnetRef:         commondomain.Reference{Resource: "subnet/sn1"},
			SkuRef:            commondomain.Reference{Resource: "nic-sku/small"},
			PublicIpRefs:      []commondomain.Reference{{Resource: "public-ip/ip1"}},
			SecurityGroupRefs: []commondomain.Reference{{Resource: "security-group/sg1"}},
		},
	}
	in.Name = "nic1"
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Provider = nicdom.ProviderID
	in.Region = "r1"
	in.Status = &nicdom.NicStatus{
		Status:       commondomain.Status{State: commondomain.ResourceStateActive},
		MacAddress:   "aa:bb:cc:dd:ee:ff",
		Addresses:    []string{"10.0.0.5"},
		PublicIpRefs: []commondomain.Reference{{Resource: "public-ip/ip1"}},
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := NicToCR(in)
	require.NoError(t, err)

	out, err := NicFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Region, out.Region)
	require.Equal(t, in.Spec.Addresses, out.Spec.Addresses)
	require.Equal(t, in.Spec.SubnetRef, out.Spec.SubnetRef)
	require.Equal(t, in.Spec.SkuRef, out.Spec.SkuRef)
	require.Equal(t, in.Spec.PublicIpRefs, out.Spec.PublicIpRefs)
	require.Equal(t, in.Spec.SecurityGroupRefs, out.Spec.SecurityGroupRefs)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
	require.Equal(t, in.Status.MacAddress, out.Status.MacAddress)
	require.Equal(t, in.Status.Addresses, out.Status.Addresses)
	require.Equal(t, in.Status.PublicIpRefs, out.Status.PublicIpRefs)
}

func TestNicToCR_UnsetSkuRef(t *testing.T) {
	in := &nicdom.Nic{
		Spec: nicdom.NicSpec{
			Addresses: []string{"10.0.0.5"},
			SubnetRef: commondomain.Reference{Resource: "subnet/sn1"},
		},
	}
	in.Name = "nic1"

	cr, err := NicToCR(in)
	require.NoError(t, err)

	out, err := NicFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, commondomain.Reference{}, out.Spec.SkuRef)
}
