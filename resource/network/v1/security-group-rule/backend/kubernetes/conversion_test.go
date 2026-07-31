package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
)

func TestSecurityGroupRuleConversionRoundTrip(t *testing.T) {
	in := &securitygroupruledom.SecurityGroupRule{
		Spec: securitygroupruledom.SecurityGroupRuleSpec{
			Direction: "ingress",
			Protocol:  "tcp",
			Version:   commondomain.IPVersionIPv4,
			Icmp:      &securitygroupruledom.IcmpConfig{Code: 1, Type: 2},
			Ports:     &securitygroupruledom.Ports{From: 80, To: 443, List: []int{22, 8080}},
			// TODO_TEST_238_239
			// SourceRef: []commondomain.Reference{{Resource: "networks/net1"}},
			SourceRef: []commondomain.Reference{{Resource: "networks/net1"}},
		},
	}
	in.Name = "sgr1"
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Provider = securitygroupruledom.ProviderID
	in.Region = "r1"
	in.Status = &securitygroupruledom.SecurityGroupRuleStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := SecurityGroupRuleToCR(in)
	require.NoError(t, err)

	out, err := SecurityGroupRuleFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Region, out.Region)
	require.Equal(t, in.Spec.Direction, out.Spec.Direction)
	require.Equal(t, in.Spec.Protocol, out.Spec.Protocol)
	require.Equal(t, in.Spec.Version, out.Spec.Version)
	require.Equal(t, in.Spec.Icmp, out.Spec.Icmp)
	require.Equal(t, in.Spec.Ports, out.Spec.Ports)
	require.Len(t, out.Spec.SourceRef, 1)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
}

func TestSecurityGroupRuleToCR_MinimalSpec(t *testing.T) {
	in := &securitygroupruledom.SecurityGroupRule{
		Spec: securitygroupruledom.SecurityGroupRuleSpec{
			Direction: "egress",
		},
	}
	in.Name = "sgr1"

	cr, err := SecurityGroupRuleToCR(in)
	require.NoError(t, err)

	out, err := SecurityGroupRuleFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, "egress", out.Spec.Direction)
	require.Nil(t, out.Spec.Icmp)
	require.Nil(t, out.Spec.Ports)
	require.Equal(t, commondomain.ResourceStatePending, out.Status.Conditions[0].State)
}
