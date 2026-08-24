package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

func TestInstanceFromAPIToAPIRoundTrip(t *testing.T) {
	// Reference.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	primaryNic := sdkschema.Reference{Resource: "nics/nic1"}
	securityGroup := sdkschema.Reference{Resource: "security-groups/sg1"}
	sdk := sdkschema.Instance{
		Spec: sdkschema.InstanceSpec{
			AntiAffinityGroup: "aag1",
			BootVolume: sdkschema.VolumeReference{
				DeviceRef: sdkschema.Reference{Resource: "block-storages/boot"},
				Type:      "virtio",
			},
			DataVolumes: []sdkschema.VolumeReference{
				{DeviceRef: sdkschema.Reference{Resource: "block-storages/data1"}, Type: "virtio"},
			},
			AdditionalNicRefs: []sdkschema.Reference{{Resource: "nics/nic2"}},
			PrimaryNicRef:     &primaryNic,
			SecurityGroupRef:  &securityGroup,
			SkuRef:            sdkschema.Reference{Resource: "skus/small"},
			SshKeys:           []string{"key-ref-1"},
			UserData:          "#cloud-config",
			Zone:              "zone-a",
		},
	}
	id := &resource.Identity{Name: "inst1", Scope: resource.Scope{Tenant: "t1", Workspace: "w1"}}

	dom, err := instanceFromAPI(sdk, id, "r1")
	require.NoError(t, err)
	require.Equal(t, "inst1", dom.Name)
	require.Equal(t, "t1", dom.Tenant)
	require.Equal(t, "w1", dom.Workspace)
	require.Equal(t, "r1", dom.Region)
	require.Equal(t, instancedom.ProviderID, dom.Provider)
	require.Equal(t, "aag1", dom.Spec.AntiAffinityGroup)
	require.Equal(t, "block-storages/boot", dom.Spec.BootVolume.DeviceRef.Resource)
	require.Len(t, dom.Spec.DataVolumes, 1)
	require.Len(t, dom.Spec.AdditionalNicRefs, 1)
	require.NotNil(t, dom.Spec.PrimaryNicRef)
	require.Equal(t, "nics/nic1", dom.Spec.PrimaryNicRef.Resource)
	require.NotNil(t, dom.Spec.SecurityGroupRef)
	require.Equal(t, "skus/small", dom.Spec.SkuRef.Resource)
	require.Equal(t, "zone-a", dom.Spec.Zone)

	out := instanceToAPIWithVerb(http.MethodPut)(dom)
	require.Equal(t, http.MethodPut, out.Metadata.Verb)
	require.Equal(t, "inst1", out.Metadata.Name)
	require.Equal(t, "zone-a", out.Spec.Zone)
	require.Equal(t, "block-storages/boot", out.Spec.BootVolume.DeviceRef.Resource)
	// metadata.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "instances/inst1", out.Metadata.Resource)
	// metadata.ref: {provider}/tenants/{tenant}/workspaces/{workspace}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "seca.compute/v1/tenants/t1/workspaces/w1/instances/inst1", out.Metadata.Ref)
}

func TestInstanceIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := instanceIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "instances", iter.Metadata.Resource)
	require.Equal(t, "seca.compute/v1", iter.Metadata.Provider)
}

func TestInstanceToAPI_Status(t *testing.T) {
	dom := &instancedom.Instance{
		Spec: instancedom.InstanceSpec{Zone: "zone-a"},
	}
	dom.Name = "inst1"

	out := instanceToAPIWithVerb(http.MethodGet)(dom)
	require.Nil(t, out.Status)

	dom.Status = &instancedom.InstanceStatus{
		Status:     commondomain.Status{State: commondomain.ResourceStateActive},
		PowerState: instancedom.PowerStateOn,
	}

	out = instanceToAPIWithVerb(http.MethodGet)(dom)
	require.NotNil(t, out.Status)
	require.Equal(t, sdkschema.InstanceStatusPowerStateOn, out.Status.PowerState)
}
