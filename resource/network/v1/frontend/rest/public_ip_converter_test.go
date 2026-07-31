package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

func TestPublicIpFromAPIToAPIRoundTrip(t *testing.T) {
	sdk := sdkschema.PublicIp{
		Spec: sdkschema.PublicIpSpec{
			Address: "203.0.113.5",
			Version: sdkschema.IPVersionIPv4,
		},
	}
	id := &resource.Identity{Name: "ip1", Scope: resource.Scope{Tenant: "t1", Workspace: "w1"}}

	dom := publicIpFromAPI(sdk, id, "r1")
	require.Equal(t, "ip1", dom.Name)
	require.Equal(t, "t1", dom.Tenant)
	require.Equal(t, "w1", dom.Workspace)
	require.Equal(t, "r1", dom.Region)
	require.Equal(t, publicipdom.ProviderID, dom.Provider)
	require.Equal(t, "203.0.113.5", dom.Spec.Address)
	require.Equal(t, commondomain.IPVersionIPv4, dom.Spec.Version)

	out := publicIpToAPIWithVerb(http.MethodPut)(dom)
	require.Equal(t, http.MethodPut, out.Metadata.Verb)
	require.Equal(t, "ip1", out.Metadata.Name)
	require.Equal(t, "203.0.113.5", out.Spec.Address)
	require.Equal(t, sdkschema.IPVersionIPv4, out.Spec.Version)
	// TODO_TEST_238_239
	// require.Equal(t, "public-ip/ip1", out.Metadata.Resource)
	require.Equal(t, "public-ips/ip1", out.Metadata.Resource)
	// TODO_TEST_238_239
	// require.Equal(t, "seca.network/v1/tenants/t1/workspaces/w1/providers/public-ip/ip1", out.Metadata.Ref)
	require.Equal(t, "seca.network/v1/tenants/t1/workspaces/w1/public-ips/ip1", out.Metadata.Ref)
}

func TestPublicIpIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := publicIpIteratorToAPI(nil, nil)
	// TODO_TEST_238_239
	// require.Equal(t, "public-ips", iter.Metadata.Resource)
	require.Equal(t, "public-ips", iter.Metadata.Resource)
	require.Equal(t, "seca.network/v1", iter.Metadata.Provider)
}

func TestPublicIpToAPI_AttachedTo(t *testing.T) {
	dom := &publicipdom.PublicIp{
		Spec: publicipdom.PublicIpSpec{Version: commondomain.IPVersionIPv6},
	}
	dom.Name = "ip1"

	out := publicIpToAPIWithVerb(http.MethodGet)(dom)
	require.Nil(t, out.Status)

	// TODO_TEST_238_239
	// attachedTo := commondomain.Reference{Resource: "nic/nic1"}
	attachedTo := commondomain.Reference{Resource: "nics/nic1"}
	dom.Status = &publicipdom.PublicIpStatus{
		Status:     commondomain.Status{State: commondomain.ResourceStateActive},
		AttachedTo: &attachedTo,
		IpAddress:  "203.0.113.5",
	}

	out = publicIpToAPIWithVerb(http.MethodGet)(dom)
	require.NotNil(t, out.Status)
	require.NotNil(t, out.Status.AttachedTo)
	require.Equal(t, "nics/nic1", out.Status.AttachedTo.Resource)
	require.Equal(t, "203.0.113.5", out.Status.IpAddress)
}
