package converter_test

import (
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"

	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	securitygroup "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
)

func vpcRef() v1alpha1.ResourceReference {
	return v1alpha1.ResourceReference{Name: "my-network", Namespace: "ws-ns"}
}

func projectRef() v1alpha1.ResourceReference {
	return v1alpha1.ResourceReference{Name: "test-workspace", Namespace: "tn-ns"}
}

func TestBuildKeyPair(t *testing.T) {
	inst := &instancedom.Instance{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "vm-1"},
			Scope:          res.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
			Region:         "ITBG-Bergamo",
		},
	}

	kp := converter.BuildKeyPair(inst, "ssh-rsa AAAAKEY")
	require.Equal(t, "vm-1-key", kp.Name)
	require.Equal(t, "ssh-rsa AAAAKEY", kp.Spec.Value)
	require.Equal(t, "test-tenant", kp.Spec.Tenant)
	require.Equal(t, "ITBG-Bergamo", kp.Spec.Region)
	require.Equal(t, "test-workspace", kp.Spec.ProjectReference.Name)
}

func TestBuildSecurityGroup(t *testing.T) {
	sg := converter.BuildSecurityGroup("web", "my-network", "", "test-tenant", "ws-ns", nil, vpcRef(), projectRef())
	// Name encodes the network so the same SECA group can be materialised in several VPCs.
	require.Equal(t, "web-my-network", sg.Name)
	require.Equal(t, "ws-ns", sg.Namespace)
	require.Equal(t, "ITBG-Bergamo", sg.Spec.Region) // defaulted
	require.Equal(t, vpcRef(), sg.Spec.VPCReference)
	require.Equal(t, projectRef(), sg.Spec.ProjectReference)
}

func TestBuildSecurityRules_expansion(t *testing.T) {
	sgName := converter.MaterializedSecurityGroupName("web", "my-network")

	t.Run("port list and tcp+udp fan out; default source is all traffic", func(t *testing.T) {
		rules := converter.NormalizeInlineRules([]securitygroup.SecurityGroupRuleSpec{
			{
				Direction: "ingress",
				Protocol:  "tcp+udp",
				Ports:     &securitygroup.Ports{List: []int{80, 443}},
			},
		}, nil)

		out := converter.BuildSecurityRules(rules, sgName, "", "test-tenant", "ws-ns", vpcRef(), projectRef())
		// 2 protocols x 2 ports x 1 (default) target = 4 rules.
		require.Len(t, out, 4)

		protocols := map[string]int{}
		ports := map[string]int{}
		for i, r := range out {
			require.Equal(t, "Ingress", r.Spec.Direction)
			require.Equal(t, sgName, r.Spec.SecurityGroupReference.Name)
			require.Equal(t, vpcRef(), r.Spec.VPCReference)
			require.Equal(t, v1alpha1.SecurityRuleTarget{Type: "Ip", Value: "0.0.0.0/0"}, r.Spec.Target)
			require.Equalf(t, sgName+"-r"+itoa(i), r.Name, "rule names must be deterministic")
			protocols[r.Spec.Protocol]++
			ports[r.Spec.Port]++
		}
		require.Equal(t, 2, protocols["TCP"])
		require.Equal(t, 2, protocols["UDP"])
		require.Equal(t, 2, ports["80"])
		require.Equal(t, 2, ports["443"])
	})

	t.Run("icmp ignores ports; source is a security group", func(t *testing.T) {
		rules := converter.NormalizeInlineRules([]securitygroup.SecurityGroupRuleSpec{
			{
				Direction: "egress",
				Protocol:  "icmp",
				Icmp:      &securitygroup.IcmpConfig{Type: 8},
				Ports:     &securitygroup.Ports{From: 80, To: 90},
				SourceRef: []commondomain.Reference{{Resource: "security-groups/frontend"}},
			},
		}, nil)

		out := converter.BuildSecurityRules(rules, sgName, "ITBG-Bergamo", "test-tenant", "ws-ns", vpcRef(), projectRef())
		require.Len(t, out, 1)
		require.Equal(t, "ICMP", out[0].Spec.Protocol)
		require.Equal(t, "ALL", out[0].Spec.Port) // ports do not apply to ICMP
		require.Equal(t, "Egress", out[0].Spec.Direction)
		require.Equal(t, v1alpha1.SecurityRuleTarget{Type: "SecurityGroup", Value: "frontend"}, out[0].Spec.Target)
	})

	t.Run("port range and empty protocol map to a range and ALL", func(t *testing.T) {
		rules := converter.NormalizeInlineRules([]securitygroup.SecurityGroupRuleSpec{
			{Direction: "ingress", Ports: &securitygroup.Ports{From: 8000, To: 8100}},
		}, nil)

		out := converter.BuildSecurityRules(rules, sgName, "", "test-tenant", "ws-ns", vpcRef(), projectRef())
		require.Len(t, out, 1)
		require.Equal(t, "ALL", out[0].Spec.Protocol)
		require.Equal(t, "8000-8100", out[0].Spec.Port)
	})
}

func TestBuildCloudServer(t *testing.T) {
	inst := &instancedom.Instance{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "vm-1"},
			Scope:          res.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
		},
		Spec: instancedom.InstanceSpec{Zone: "ITBG-2"},
	}

	cs := converter.BuildCloudServer(inst, converter.CloudServerRefs{
		FlavorName: "n1.small",
		// The zone comes from the resolved boot volume, not from the instance: Aruba requires the two
		// to share one, and the handler refuses a conflicting instance zone before reaching here.
		Zone:                    "ITBG-2",
		VPCReference:            vpcRef(),
		SubnetReferences:        []v1alpha1.ResourceReference{{Name: "sub-1", Namespace: "net-ns"}},
		SecurityGroupReferences: []v1alpha1.ResourceReference{{Name: "web-my-network", Namespace: "ws-ns"}},
		KeyPairReference:        v1alpha1.ResourceReference{Name: "vm-1-key", Namespace: "ws-ns"},
		BootVolumeReference:     v1alpha1.ResourceReference{Name: "boot", Namespace: "ws-ns"},
		ProjectReference:        projectRef(),
	})

	require.Equal(t, "vm-1", cs.Name)
	require.Equal(t, "n1.small", cs.Spec.FlavorName)
	require.Equal(t, "ITBG-2", cs.Spec.Zone)
	require.Equal(t, "ITBG-Bergamo", cs.Spec.Region) // defaulted
	require.Len(t, cs.Spec.SubnetReferences, 1)
	require.Len(t, cs.Spec.SecurityGroupReferences, 1)
	require.Equal(t, "vm-1-key", cs.Spec.KeyPairReference.Name)
}

func itoa(i int) string {
	return []string{"0", "1", "2", "3", "4", "5"}[i]
}
