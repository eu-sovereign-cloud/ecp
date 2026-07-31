package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

func TestSecurityGroupRuleFromAPIToAPIRoundTrip(t *testing.T) {
	sdk := sdkschema.SecurityGroupRule{
		Spec: sdkschema.SecurityGroupRuleSpec{
			Direction: sdkschema.SecurityGroupRuleSpecDirection("ingress"),
			Protocol:  sdkschema.SecurityGroupRuleSpecProtocol("tcp"),
			Ports:     &sdkschema.Ports{From: 80, To: 443},
		},
	}
	id := &SecurityGroupRuleIdentity{name: "sgr1", tenant: "t1", workspace: "w1"}

	dom := securityGroupRuleFromAPI(sdk, id, "r1")
	require.Equal(t, "sgr1", dom.Name)
	require.Equal(t, "t1", dom.Tenant)
	require.Equal(t, "w1", dom.Workspace)
	require.Equal(t, "r1", dom.Region)
	require.Equal(t, securitygroupruledom.ProviderID, dom.Provider)
	require.Equal(t, "ingress", dom.Spec.Direction)
	require.Equal(t, "tcp", dom.Spec.Protocol)
	require.Equal(t, 80, dom.Spec.Ports.From)

	out := securityGroupRuleToAPIWithVerb(http.MethodPut)(dom)
	require.Equal(t, http.MethodPut, out.Metadata.Verb)
	require.Equal(t, "sgr1", out.Metadata.Name)
	require.Equal(t, sdkschema.SecurityGroupRuleSpecDirection("ingress"), out.Spec.Direction)
}

func TestSecurityGroupRuleIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := securityGroupRuleIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "security-group-rules", iter.Metadata.Resource)
	require.Equal(t, "seca.network/v1", iter.Metadata.Provider)
}

func TestSecurityGroupRuleToAPI_Status(t *testing.T) {
	dom := &securitygroupruledom.SecurityGroupRule{
		Spec: securitygroupruledom.SecurityGroupRuleSpec{Direction: "egress"},
	}
	dom.Name = "sgr1"

	out := securityGroupRuleToAPIWithVerb(http.MethodGet)(dom)
	require.Nil(t, out.Status)

	dom.Status = &securitygroupruledom.SecurityGroupRuleStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
	}
	out = securityGroupRuleToAPIWithVerb(http.MethodGet)(dom)
	require.NotNil(t, out.Status)
	require.Equal(t, sdkschema.ResourceStateActive, out.Status.State)
}
