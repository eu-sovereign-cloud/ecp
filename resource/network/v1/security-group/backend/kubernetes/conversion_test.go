package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
)

func TestSecurityGroupConversionRoundTrip(t *testing.T) {
	in := &securitygroupdom.SecurityGroup{
		Spec: securitygroupdom.SecurityGroupSpec{
			// TODO_TEST_238_239
			// RuleRefs: []commondomain.Reference{{Resource: "security-group-rules/rule1"}},
			RuleRefs: []commondomain.Reference{{Resource: "security-group-rules/rule1"}},
			Rules: []securitygroupdom.SecurityGroupRuleSpec{
				{
					Direction: "ingress",
					Protocol:  "tcp",
					Version:   commondomain.IPVersionIPv4,
					Ports:     &securitygroupdom.Ports{From: 80, To: 443},
					// TODO_TEST_238_239
					// SourceRef: []commondomain.Reference{{Resource: "networks/net1"}},
					SourceRef: []commondomain.Reference{{Resource: "networks/net1"}},
				},
			},
		},
	}
	in.Name = "sg1"
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Provider = securitygroupdom.ProviderID
	in.Region = "r1"
	in.Status = &securitygroupdom.SecurityGroupStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
		Rules: []securitygroupdom.SecurityGroupRuleStatus{
			{State: commondomain.ResourceStateActive},
		},
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := SecurityGroupToCR(in)
	require.NoError(t, err)

	out, err := SecurityGroupFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Region, out.Region)
	require.Len(t, out.Spec.RuleRefs, 1)
	require.Len(t, out.Spec.Rules, 1)
	require.Equal(t, in.Spec.Rules[0].Direction, out.Spec.Rules[0].Direction)
	require.Equal(t, in.Spec.Rules[0].Ports, out.Spec.Rules[0].Ports)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
	require.Len(t, out.Status.Rules, 1)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.Rules[0].State)
}

func TestSecurityGroupToCR_EmptySpec(t *testing.T) {
	in := &securitygroupdom.SecurityGroup{}
	in.Name = "sg1"

	cr, err := SecurityGroupToCR(in)
	require.NoError(t, err)

	out, err := SecurityGroupFromCR(cr)
	require.NoError(t, err)
	require.Empty(t, out.Spec.Rules)
	require.Equal(t, commondomain.ResourceStatePending, out.Status.Conditions[0].State)
}
