package kubernetes

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"

	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// volumeReferenceFromCR converts a schemav1.VolumeReference into an instancedom.VolumeReference.
func volumeReferenceFromCR(ref schemav1.VolumeReference) instancedom.VolumeReference {
	return instancedom.VolumeReference{
		DeviceRef: commonbackend.ReferenceFromCR(ref.DeviceRef),
		Type:      string(ref.Type),
	}
}

// volumeReferenceToCR converts an instancedom.VolumeReference into a schemav1.VolumeReference.
func volumeReferenceToCR(ref instancedom.VolumeReference) schemav1.VolumeReference {
	return schemav1.VolumeReference{
		DeviceRef: commonbackend.ReferenceToCR(ref.DeviceRef),
		Type:      schemav1.VolumeReferenceType(ref.Type),
	}
}

// InstanceFromCR converts either a concrete *Instance or *unstructured.Unstructured into an *instancedom.Instance.
func InstanceFromCR(obj client.Object) (*instancedom.Instance, error) {
	var cr Instance

	switch t := obj.(type) {
	case *Instance:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Instance: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	spec := instancedom.InstanceSpec{
		AntiAffinityGroup: cr.Spec.AntiAffinityGroup,
		BootVolume:        volumeReferenceFromCR(cr.Spec.BootVolume),
		SkuRef:            commonbackend.ReferenceFromCR(cr.Spec.SkuRef),
		SshKeys:           cr.Spec.SshKeys,
		UserData:          cr.Spec.UserData,
		Zone:              cr.Spec.Zone,
	}
	for _, r := range cr.Spec.AdditionalNicRefs {
		spec.AdditionalNicRefs = append(spec.AdditionalNicRefs, commonbackend.ReferenceFromCR(r))
	}
	for _, v := range cr.Spec.DataVolumes {
		spec.DataVolumes = append(spec.DataVolumes, volumeReferenceFromCR(v))
	}
	if cr.Spec.PrimaryNicRef != nil {
		ref := commonbackend.ReferenceFromCR(*cr.Spec.PrimaryNicRef)
		spec.PrimaryNicRef = &ref
	}
	if cr.Spec.SecurityGroupRef != nil {
		ref := commonbackend.ReferenceFromCR(*cr.Spec.SecurityGroupRef)
		spec.SecurityGroupRef = &ref
	}

	inst := &instancedom.Instance{Spec: spec}
	inst.Name = cr.GetName()
	inst.ResourceVersion = cr.GetResourceVersion()
	inst.CreatedAt = cr.GetCreationTimestamp().Time
	inst.UpdatedAt = cr.GetCreationTimestamp().Time
	inst.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	inst.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	inst.Workspace = internalLabels[k8slabels.InternalWorkspaceLabel]
	inst.Region = internalLabels[k8slabels.InternalRegionLabel]
	inst.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	inst.Annotations = cr.CommonData.Annotations
	inst.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		inst.DeletedAt = &ts.Time
	}

	inst.Status = &instancedom.InstanceStatus{}
	if cr.Status != nil {
		inst.Status.State = commonbackend.ResourceStateFromCR(cr.Status.State)
		inst.Status.Conditions = commonbackend.ConditionsFromCR(cr.Status.Conditions)
		inst.Status.PowerState = instancedom.PowerState(cr.Status.PowerState)
		if cr.Status.PowerStateSince != nil {
			inst.Status.PowerStateSince = &cr.Status.PowerStateSince.Time
		}
	} else {
		inst.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return inst, nil
}

// InstanceToCR converts an *instancedom.Instance to a Kubernetes Instance CR.
func InstanceToCR(inst *instancedom.Instance) (client.Object, error) {
	if inst == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(inst.Labels)
	crLabels[k8slabels.InternalTenantLabel] = inst.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = inst.Workspace
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(inst.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = inst.Region

	additionalNicRefs := make([]schemav1.Reference, len(inst.Spec.AdditionalNicRefs))
	for i, r := range inst.Spec.AdditionalNicRefs {
		additionalNicRefs[i] = commonbackend.ReferenceToCR(r)
	}
	dataVolumes := make([]schemav1.VolumeReference, len(inst.Spec.DataVolumes))
	for i, v := range inst.Spec.DataVolumes {
		dataVolumes[i] = volumeReferenceToCR(v)
	}

	spec := InstanceSpec{
		AdditionalNicRefs: additionalNicRefs,
		AntiAffinityGroup: inst.Spec.AntiAffinityGroup,
		BootVolume:        volumeReferenceToCR(inst.Spec.BootVolume),
		DataVolumes:       dataVolumes,
		SkuRef:            commonbackend.ReferenceToCR(inst.Spec.SkuRef),
		SshKeys:           inst.Spec.SshKeys,
		UserData:          inst.Spec.UserData,
		Zone:              inst.Spec.Zone,
	}
	if inst.Spec.PrimaryNicRef != nil {
		ref := commonbackend.ReferenceToCR(*inst.Spec.PrimaryNicRef)
		spec.PrimaryNicRef = &ref
	}
	if inst.Spec.SecurityGroupRef != nil {
		ref := commonbackend.ReferenceToCR(*inst.Spec.SecurityGroupRef)
		spec.SecurityGroupRef = &ref
	}

	cr := &Instance{
		ObjectMeta: v1.ObjectMeta{
			Name:            inst.Name,
			Namespace:       k8sadapter.ComputeNamespace(inst),
			Labels:          crLabels,
			ResourceVersion: inst.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: inst.Annotations,
			Extensions:  inst.Extensions,
			Labels:      slices.Collect(maps.Keys(inst.Labels)),
		},
		Spec: spec,
	}
	cr.SetGroupVersionKind(InstanceGVK)

	if inst.Status != nil && len(inst.Status.Conditions) > 0 {
		state := commonbackend.ResourceStateToCR(inst.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}
		powerState := inst.Status.PowerState
		if powerState == "" {
			powerState = instancedom.PowerStateOff
		}
		cr.Status = &InstanceStatus{
			Conditions: commonbackend.ConditionsToCR(inst.Status.Conditions),
			State:      *state,
			PowerState: InstanceStatusPowerState(powerState),
		}
		if inst.Status.PowerStateSince != nil {
			cr.Status.PowerStateSince = &v1.Time{Time: *inst.Status.PowerStateSince}
		}
	}

	return cr, nil
}
