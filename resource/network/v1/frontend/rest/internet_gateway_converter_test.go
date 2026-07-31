package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

func TestInternetGatewayFromAPIToAPIRoundTrip(t *testing.T) {
	sdk := sdkschema.InternetGateway{
		Spec: sdkschema.InternetGatewaySpec{
			EgressOnly: true,
		},
	}
	id := &InternetGatewayIdentity{name: "ig1", tenant: "t1", workspace: "w1"}

	dom := internetGatewayFromAPI(sdk, id, "r1")
	require.Equal(t, "ig1", dom.Name)
	require.Equal(t, "t1", dom.Tenant)
	require.Equal(t, "w1", dom.Workspace)
	require.Equal(t, "r1", dom.Region)
	require.Equal(t, internetgatewaydom.ProviderID, dom.Provider)
	require.True(t, dom.Spec.EgressOnly)

	out := internetGatewayToAPIWithVerb(http.MethodPut)(dom)
	require.Equal(t, http.MethodPut, out.Metadata.Verb)
	require.Equal(t, "ig1", out.Metadata.Name)
	require.True(t, out.Spec.EgressOnly)
	// metadata.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "internet-gateways/ig1", out.Metadata.Resource)
	// metadata.ref: {provider}/tenants/{tenant}/workspaces/{workspace}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "seca.network/v1/tenants/t1/workspaces/w1/internet-gateways/ig1", out.Metadata.Ref)
}

func TestInternetGatewayIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := internetGatewayIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "internet-gateways", iter.Metadata.Resource)
	require.Equal(t, "seca.network/v1", iter.Metadata.Provider)
}

func TestInternetGatewayToAPI_Status(t *testing.T) {
	dom := &internetgatewaydom.InternetGateway{
		Spec: internetgatewaydom.InternetGatewaySpec{EgressOnly: false},
	}
	dom.Name = "ig1"

	out := internetGatewayToAPIWithVerb(http.MethodGet)(dom)
	require.Nil(t, out.Status)

	dom.Status = &internetgatewaydom.InternetGatewayStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
	}
	out = internetGatewayToAPIWithVerb(http.MethodGet)(dom)
	require.NotNil(t, out.Status)
	require.Equal(t, sdkschema.ResourceStateActive, out.Status.State)
}
