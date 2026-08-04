package rest

import (
	"strconv"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// instanceListParamsFromAPI converts SDK ListInstancesParams to resource.ListParams.
func instanceListParamsFromAPI(params sdkcompute.ListInstancesParams, tenant, workspace string) resource.ListParams {
	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}
	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}
	return resource.ListParams{
		Scope:     resource.Scope{Tenant: tenant, Workspace: workspace},
		Limit:     validation.GetLimit(params.Limit),
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// volumeReferenceToAPI converts an instancedom.VolumeReference to its SDK representation.
func volumeReferenceToAPI(v instancedom.VolumeReference) sdkschema.VolumeReference {
	return sdkschema.VolumeReference{
		DeviceRef: commonfrontend.ReferenceToAPI(v.DeviceRef),
		Type:      sdkschema.VolumeReferenceType(v.Type),
	}
}

// volumeReferenceFromAPI converts an SDK VolumeReference to an instancedom.VolumeReference.
func volumeReferenceFromAPI(v sdkschema.VolumeReference) instancedom.VolumeReference {
	return instancedom.VolumeReference{
		DeviceRef: commonfrontend.ReferenceFromAPI(v.DeviceRef),
		Type:      string(v.Type),
	}
}

// instanceToAPIWithVerb returns a func that converts an Instance to its SDK representation with the given verb.
func instanceToAPIWithVerb(verb string) func(inst *instancedom.Instance) *sdkschema.Instance {
	return func(inst *instancedom.Instance) *sdkschema.Instance {
		sdk := instanceToAPI(inst)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

func instanceToAPI(inst *instancedom.Instance) *sdkschema.Instance {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(inst.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	const kind = sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindInstance

	out := &sdkschema.Instance{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:      instancedom.Version,
			CreatedAt:       inst.CreatedAt,
			LastModifiedAt:  inst.UpdatedAt,
			Kind:            kind,
			Name:            inst.Name,
			Tenant:          inst.Tenant,
			Workspace:       inst.Workspace,
			Provider:        inst.Provider,
			Region:          inst.Region,
			Resource:        commondomain.FormatRegionalResource(kind, inst.Name),
			Ref:             commondomain.FormatRegionalWorkspaceScopedRef(inst.Provider, inst.Tenant, inst.Workspace, kind, inst.Name),
			ResourceVersion: resourceVersion,
		},
		Labels:      inst.Labels,
		Annotations: inst.Annotations,
		Extensions:  inst.Extensions,
		Spec: sdkschema.InstanceSpec{
			AntiAffinityGroup: inst.Spec.AntiAffinityGroup,
			BootVolume:        volumeReferenceToAPI(inst.Spec.BootVolume),
			SkuRef:            commonfrontend.ReferenceToAPI(inst.Spec.SkuRef),
			SshKeys:           inst.Spec.SshKeys,
			UserData:          inst.Spec.UserData,
			Zone:              inst.Spec.Zone,
		},
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}
	for _, r := range inst.Spec.AdditionalNicRefs {
		out.Spec.AdditionalNicRefs = append(out.Spec.AdditionalNicRefs, commonfrontend.ReferenceToAPI(r))
	}
	for _, v := range inst.Spec.DataVolumes {
		out.Spec.DataVolumes = append(out.Spec.DataVolumes, volumeReferenceToAPI(v))
	}
	out.Spec.PrimaryNicRef = commonfrontend.ReferencePtrToAPI(inst.Spec.PrimaryNicRef)
	out.Spec.SecurityGroupRef = commonfrontend.ReferencePtrToAPI(inst.Spec.SecurityGroupRef)

	if inst.Status != nil {
		out.Status = &sdkschema.InstanceStatus{
			Conditions:      commonfrontend.ConditionsToAPI(inst.Status.Conditions),
			State:           commonfrontend.ResourceStateToAPI(inst.Status.State),
			PowerState:      sdkschema.InstanceStatusPowerState(inst.Status.PowerState),
			PowerStateSince: inst.Status.PowerStateSince,
		}
	}
	if inst.DeletedAt != nil {
		out.Metadata.DeletedAt = inst.DeletedAt
	}
	return out
}

// instanceIteratorToAPI converts a list of Instance to an SDK InstanceIterator.
func instanceIteratorToAPI(insts []*instancedom.Instance, nextSkipToken *string) *sdkcompute.InstanceIterator {
	items := make([]sdkschema.Instance, len(insts))
	for i := range insts {
		items[i] = *instanceToAPI(insts[i])
	}
	iterator := &sdkcompute.InstanceIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: instancedom.ProviderID,
			Resource: instancedom.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator
}

// instanceFromAPI converts an SDK Instance to an Instance.
func instanceFromAPI(sdk sdkschema.Instance, id *resource.Identity, region string) *instancedom.Instance {
	inst := &instancedom.Instance{
		Spec: instancedom.InstanceSpec{
			AntiAffinityGroup: sdk.Spec.AntiAffinityGroup,
			BootVolume:        volumeReferenceFromAPI(sdk.Spec.BootVolume),
			SkuRef:            commonfrontend.ReferenceFromAPI(sdk.Spec.SkuRef),
			SshKeys:           sdk.Spec.SshKeys,
			UserData:          sdk.Spec.UserData,
			Zone:              sdk.Spec.Zone,
		},
	}
	for _, r := range sdk.Spec.AdditionalNicRefs {
		inst.Spec.AdditionalNicRefs = append(inst.Spec.AdditionalNicRefs, commonfrontend.ReferenceFromAPI(r))
	}
	for _, v := range sdk.Spec.DataVolumes {
		inst.Spec.DataVolumes = append(inst.Spec.DataVolumes, volumeReferenceFromAPI(v))
	}
	if sdk.Spec.PrimaryNicRef != nil {
		ref := commonfrontend.ReferenceFromAPI(*sdk.Spec.PrimaryNicRef)
		inst.Spec.PrimaryNicRef = &ref
	}
	if sdk.Spec.SecurityGroupRef != nil {
		ref := commonfrontend.ReferenceFromAPI(*sdk.Spec.SecurityGroupRef)
		inst.Spec.SecurityGroupRef = &ref
	}

	inst.Name = id.GetName()
	inst.ResourceVersion = id.GetVersion()
	inst.Provider = instancedom.ProviderID
	inst.Tenant = id.GetTenant()
	inst.Workspace = id.GetWorkspace()
	inst.Region = region
	inst.Labels = sdk.Labels
	inst.Annotations = sdk.Annotations
	inst.Extensions = sdk.Extensions

	return inst
}

// newInstanceWithIdentity returns an *instancedom.Instance populated with identity fields from ir.
func newInstanceWithIdentity(ir persistence.IdentifiableResource) *instancedom.Instance {
	d := &instancedom.Instance{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	return d
}
