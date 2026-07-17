package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

func TestSecurityGroupFromAPIToAPIRoundTrip(t *testing.T) {
	sdk := sdkschema.SecurityGroup{
		Spec: sdkschema.SecurityGroupSpec{
			RuleRefs: []sdkschema.Reference{{Resource: "security-group-rules/rule1"}},
			Rules: []sdkschema.SecurityGroupRuleSpec{
				{Direction: sdkschema.SecurityGroupRuleSpecDirection("ingress"), Protocol: sdkschema.SecurityGroupRuleSpecProtocol("tcp")},
			},
		},
	}
	id := &SecurityGroupIdentity{name: "sg1", tenant: "t1", workspace: "w1"}

	dom := securityGroupFromAPI(sdk, id, "r1")
	require.Equal(t, "sg1", dom.Name)
	require.Equal(t, "t1", dom.Tenant)
	require.Equal(t, "w1", dom.Workspace)
	require.Equal(t, "r1", dom.Region)
	require.Equal(t, securitygroupdom.ProviderID, dom.Provider)
	require.Len(t, dom.Spec.RuleRefs, 1)
	require.Len(t, dom.Spec.Rules, 1)
	require.Equal(t, "ingress", dom.Spec.Rules[0].Direction)

	out := securityGroupToAPIWithVerb(http.MethodPut)(dom)
	require.Equal(t, http.MethodPut, out.Metadata.Verb)
	require.Equal(t, "sg1", out.Metadata.Name)
	require.Len(t, out.Spec.Rules, 1)
}

func TestSecurityGroupIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := securityGroupIteratorToAPI(nil, nil)
	require.Equal(t, "security-groups", iter.Metadata.Resource)
	require.Equal(t, "seca.network/v1", iter.Metadata.Provider)
}

func TestSecurityGroupToAPI_Status(t *testing.T) {
	dom := &securitygroupdom.SecurityGroup{}
	dom.Name = "sg1"

	out := securityGroupToAPIWithVerb(http.MethodGet)(dom)
	require.Nil(t, out.Status)

	dom.Status = &securitygroupdom.SecurityGroupStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
		Rules:  []securitygroupdom.SecurityGroupRuleStatus{{State: commondomain.ResourceStateActive}},
	}
	out = securityGroupToAPIWithVerb(http.MethodGet)(dom)
	require.NotNil(t, out.Status)
	require.Equal(t, sdkschema.ResourceStateActive, out.Status.State)
	require.Len(t, out.Status.Rules, 1)
}
