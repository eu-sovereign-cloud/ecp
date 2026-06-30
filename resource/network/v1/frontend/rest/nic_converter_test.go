package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

func TestNicFromAPIToAPIRoundTrip(t *testing.T) {
	sdk := sdkschema.Nic{
		Spec: sdkschema.NicSpec{
			Addresses: []string{"10.0.0.5"},
			SubnetRef: sdkschema.Reference{Resource: "subnet/sn1"},
			SkuRef:    &sdkschema.Reference{Resource: "nic-sku/small"},
		},
	}
	id := &NicIdentity{name: "nic1", tenant: "t1", workspace: "w1"}

	dom := nicFromAPI(sdk, id, "r1")
	require.Equal(t, "nic1", dom.Name)
	require.Equal(t, "t1", dom.Tenant)
	require.Equal(t, "w1", dom.Workspace)
	require.Equal(t, "r1", dom.Region)
	require.Equal(t, nicdom.ProviderID, dom.Provider)
	require.Equal(t, []string{"10.0.0.5"}, dom.Spec.Addresses)
	require.Equal(t, "subnet/sn1", dom.Spec.SubnetRef.Resource)
	require.Equal(t, "nic-sku/small", dom.Spec.SkuRef.Resource)

	out := nicToAPIWithVerb(http.MethodPut)(dom)
	require.Equal(t, http.MethodPut, out.Metadata.Verb)
	require.Equal(t, "nic1", out.Metadata.Name)
	require.Equal(t, []string{"10.0.0.5"}, out.Spec.Addresses)
	require.NotNil(t, out.Spec.SkuRef)
	require.Equal(t, "nic-sku/small", out.Spec.SkuRef.Resource)
	require.Equal(t, "nic/nic1", out.Metadata.Resource)
	require.Equal(t, "seca.network/v1/tenants/t1/workspaces/w1/providers/nic/nic1", out.Metadata.Ref)
}

func TestNicIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := NicIteratorToAPI(nil, nil)
	require.Equal(t, "nics", iter.Metadata.Resource)
	require.Equal(t, "seca.network/v1", iter.Metadata.Provider)
}

func TestNicFromAPI_NilSkuRef(t *testing.T) {
	sdk := sdkschema.Nic{
		Spec: sdkschema.NicSpec{
			Addresses: []string{"10.0.0.5"},
			SubnetRef: sdkschema.Reference{Resource: "subnet/sn1"},
		},
	}
	dom := nicFromAPI(sdk, &NicIdentity{name: "nic1"}, "r1")
	require.Equal(t, commondomain.Reference{}, dom.Spec.SkuRef)

	out := nicToAPIWithVerb(http.MethodPut)(dom)
	require.Nil(t, out.Spec.SkuRef)
}
