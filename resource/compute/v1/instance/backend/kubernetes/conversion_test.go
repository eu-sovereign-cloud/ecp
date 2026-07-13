package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	. "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
)

func TestInstanceConversionRoundTrip(t *testing.T) {
	primaryNic := commondomain.Reference{Resource: "nic/nic1"}
	securityGroup := commondomain.Reference{Resource: "security-group/sg1"}
	in := &instancedom.Instance{
		Spec: instancedom.InstanceSpec{
			AntiAffinityGroup: "aag1",
			BootVolume: instancedom.VolumeReference{
				DeviceRef: commondomain.Reference{Resource: "block-storage/boot"},
				Type:      "virtio",
			},
			DataVolumes: []instancedom.VolumeReference{
				{DeviceRef: commondomain.Reference{Resource: "block-storage/data1"}, Type: "virtio"},
			},
			AdditionalNicRefs: []commondomain.Reference{{Resource: "nic/nic2"}},
			PrimaryNicRef:     &primaryNic,
			SecurityGroupRef:  &securityGroup,
			SkuRef:            commondomain.Reference{Resource: "sku/small"},
			SshKeys:           []string{"key-ref-1"},
			UserData:          "#cloud-config",
			Zone:              "zone-a",
		},
	}
	in.Name = "inst1"
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
				DeviceRef: commondomain.Reference{Resource: "block-storage/boot"},
			},
			SkuRef: commondomain.Reference{Resource: "sku/small"},
			Zone:   "zone-a",
		},
	}
	in.Name = "inst1"

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

func TestInstanceToCR_PowerStateDefaultsOff(t *testing.T) {
	in := &instancedom.Instance{
		Spec: instancedom.InstanceSpec{
			BootVolume: instancedom.VolumeReference{DeviceRef: commondomain.Reference{Resource: "block-storage/boot"}},
			SkuRef:     commondomain.Reference{Resource: "sku/small"},
			Zone:       "zone-a",
		},
	}
	in.Name = "inst1"
	// Status with a condition but no explicit power state.
	in.Status = &instancedom.InstanceStatus{Status: commondomain.Status{State: commondomain.ResourceStateActive}}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := InstanceToCR(in)
	require.NoError(t, err)

	out, err := InstanceFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, instancedom.PowerStateOff, out.Status.PowerState)
}
