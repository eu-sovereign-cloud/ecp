package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	. "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
)

const (
	testInstanceName = "inst1"
	// Reference.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	testBootDevice = "block-storages/boot"
	testSku        = "skus/small"
	testZone       = "zone-a"
)

func TestInstanceConversionRoundTrip(t *testing.T) {
	// Reference.resource values use {collection}/{name}; see testBootDevice/testSku above.
	primaryNic := commondomain.Reference{Resource: "nics/nic1"}
	securityGroup := commondomain.Reference{Resource: "security-groups/sg1"}
	in := &instancedom.Instance{
		Spec: instancedom.InstanceSpec{
			AntiAffinityGroup: "aag1",
			BootVolume: instancedom.VolumeReference{
				DeviceRef: commondomain.Reference{Resource: testBootDevice},
				Type:      "virtio",
			},
			DataVolumes: []instancedom.VolumeReference{
				{DeviceRef: commondomain.Reference{Resource: "block-storages/data1"}, Type: "virtio"},
			},
			AdditionalNicRefs: []commondomain.Reference{{Resource: "nics/nic2"}},
			PrimaryNicRef:     &primaryNic,
			SecurityGroupRef:  &securityGroup,
			SkuRef:            commondomain.Reference{Resource: testSku},
			SshKeys:           []string{"key-ref-1"},
			UserData:          "#cloud-config",
			Zone:              testZone,
		},
	}
	in.Name = testInstanceName
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Provider = instancedom.ProviderID
	in.Region = "r1"
	in.Status = &instancedom.InstanceStatus{
		Status:     commondomain.Status{State: commondomain.ResourceStateActive},
		PowerState: instancedom.PowerStateOn,
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := InstanceToCR(in)
	require.NoError(t, err)

	out, err := InstanceFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Region, out.Region)
	require.Equal(t, in.Spec.AntiAffinityGroup, out.Spec.AntiAffinityGroup)
	require.Equal(t, in.Spec.BootVolume, out.Spec.BootVolume)
	require.Equal(t, in.Spec.DataVolumes, out.Spec.DataVolumes)
	require.Equal(t, in.Spec.AdditionalNicRefs, out.Spec.AdditionalNicRefs)
	require.NotNil(t, out.Spec.PrimaryNicRef)
	require.Equal(t, *in.Spec.PrimaryNicRef, *out.Spec.PrimaryNicRef)
	require.NotNil(t, out.Spec.SecurityGroupRef)
	require.Equal(t, *in.Spec.SecurityGroupRef, *out.Spec.SecurityGroupRef)
	require.Equal(t, in.Spec.SkuRef, out.Spec.SkuRef)
	require.Equal(t, in.Spec.SshKeys, out.Spec.SshKeys)
	require.Equal(t, in.Spec.UserData, out.Spec.UserData)
	require.Equal(t, in.Spec.Zone, out.Spec.Zone)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
	require.Equal(t, instancedom.PowerStateOn, out.Status.PowerState)
}

func TestInstanceToCR_OptionalFieldsUnset(t *testing.T) {
	in := &instancedom.Instance{
		Spec: instancedom.InstanceSpec{
			BootVolume: instancedom.VolumeReference{
				DeviceRef: commondomain.Reference{Resource: testBootDevice},
			},
			SkuRef: commondomain.Reference{Resource: testSku},
			Zone:   testZone,
		},
	}
	in.Name = testInstanceName

	cr, err := InstanceToCR(in)
	require.NoError(t, err)

	out, err := InstanceFromCR(cr)
	require.NoError(t, err)
	require.Nil(t, out.Spec.PrimaryNicRef)
	require.Nil(t, out.Spec.SecurityGroupRef)
	require.Empty(t, out.Spec.AdditionalNicRefs)
	require.Empty(t, out.Spec.DataVolumes)
}

func TestInstanceToCR_Nil(t *testing.T) {
	_, err := InstanceToCR(nil)
	require.Error(t, err)
}

func TestInstancePowerIntentRoundTrip(t *testing.T) {
	in := &instancedom.Instance{
		Spec: instancedom.InstanceSpec{
			BootVolume: instancedom.VolumeReference{DeviceRef: commondomain.Reference{Resource: testBootDevice}},
			SkuRef:     commondomain.Reference{Resource: testSku},
			Zone:       testZone,
		},
		DesiredPowerState: instancedom.PowerStateOn,
		RestartID:         "abc123",
		RestartPhase:      instancedom.RestartPhasePowerOff,
	}
	in.Name = testInstanceName

	cr, err := InstanceToCR(in)
	require.NoError(t, err)

	out, err := InstanceFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, instancedom.PowerStateOn, out.DesiredPowerState)
	require.Equal(t, "abc123", out.RestartID)
	require.Equal(t, instancedom.RestartPhasePowerOff, out.RestartPhase)
}

func TestInstanceToCR_NoPowerIntentByDefault(t *testing.T) {
	in := &instancedom.Instance{
		Spec: instancedom.InstanceSpec{
			BootVolume: instancedom.VolumeReference{DeviceRef: commondomain.Reference{Resource: testBootDevice}},
			SkuRef:     commondomain.Reference{Resource: testSku},
			Zone:       testZone,
		},
	}
	in.Name = testInstanceName

	cr, err := InstanceToCR(in)
	require.NoError(t, err)

	out, err := InstanceFromCR(cr)
	require.NoError(t, err)
	require.Empty(t, out.DesiredPowerState)
	require.Empty(t, out.RestartID)
	require.Empty(t, out.RestartPhase)
}

func TestInstanceToCR_PowerStateDefaultsOff(t *testing.T) {
	in := &instancedom.Instance{
		Spec: instancedom.InstanceSpec{
			BootVolume: instancedom.VolumeReference{DeviceRef: commondomain.Reference{Resource: testBootDevice}},
			SkuRef:     commondomain.Reference{Resource: testSku},
			Zone:       testZone,
		},
	}
	in.Name = testInstanceName
	// Status with a condition but no explicit power state.
	in.Status = &instancedom.InstanceStatus{Status: commondomain.Status{State: commondomain.ResourceStateActive}}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := InstanceToCR(in)
	require.NoError(t, err)

	out, err := InstanceFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, instancedom.PowerStateOff, out.Status.PowerState)
}
