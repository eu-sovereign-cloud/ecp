package rest

import (
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

// SecurityGroupIdentity carries identity for a single security group resource.
type SecurityGroupIdentity struct {
	name            string
	tenant          string
	workspace       string
	resourceVersion string
}

func (sg *SecurityGroupIdentity) GetName() string      { return sg.name }
func (sg *SecurityGroupIdentity) GetVersion() string   { return sg.resourceVersion }
func (sg *SecurityGroupIdentity) GetTenant() string    { return sg.tenant }
func (sg *SecurityGroupIdentity) GetWorkspace() string { return sg.workspace }

var _ persistence.IdentifiableResource = (*SecurityGroupIdentity)(nil)

// securityGroupListParamsFromAPI converts SDK ListSecurityGroupsParams to resource.ListParams.
func securityGroupListParamsFromAPI(params sdknetwork.ListSecurityGroupsParams, tenant, workspace string) resource.ListParams {
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

// securityGroupToAPIWithVerb returns a func that converts a SecurityGroup to its SDK representation with the given verb.
func securityGroupToAPIWithVerb(verb string) func(sg *securitygroupdom.SecurityGroup) *sdkschema.SecurityGroup {
	return func(sg *securitygroupdom.SecurityGroup) *sdkschema.SecurityGroup {
		sdk := securityGroupToAPI(sg)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

func securityGroupToAPI(sg *securitygroupdom.SecurityGroup) *sdkschema.SecurityGroup {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(sg.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	const kind = sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindSecurityGroup

	out := &sdkschema.SecurityGroup{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:      securitygroupdom.Version,
			CreatedAt:       sg.CreatedAt,
			LastModifiedAt:  sg.UpdatedAt,
			Kind:            kind,
			Name:            sg.Name,
			Tenant:          sg.Tenant,
			Workspace:       sg.Workspace,
			Provider:        sg.Provider,
			Region:          sg.Region,
			Resource:        commondomain.FormatRegionalResource(kind, sg.Name),
			Ref:             commondomain.FormatRegionalWorkspaceScopedRef(sg.Provider, sg.Tenant, sg.Workspace, kind, sg.Name),
			ResourceVersion: resourceVersion,
		},
		Labels:      sg.Labels,
		Annotations: sg.Annotations,
		Extensions:  sg.Extensions,
		Spec:        securityGroupSpecToAPI(sg.Spec),
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}

	if sg.Status != nil {
		out.Status = &sdkschema.SecurityGroupStatus{
			Conditions: commonfrontend.ConditionsToAPI(sg.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(sg.Status.State),
		}
		for _, rs := range sg.Status.Rules {
			out.Status.Rules = append(out.Status.Rules, sdkschema.SecurityGroupRuleStatus{
				Conditions: commonfrontend.ConditionsToAPI(rs.Conditions),
				State:      commonfrontend.ResourceStateToAPI(rs.State),
			})
		}
	}
	if sg.DeletedAt != nil {
		out.Metadata.DeletedAt = sg.DeletedAt
	}
	return out
}

// securityGroupIteratorToAPI converts a list of SecurityGroup to an SDK SecurityGroupIterator.
func securityGroupIteratorToAPI(sgs []*securitygroupdom.SecurityGroup, nextSkipToken *string) *sdknetwork.SecurityGroupIterator {
	items := make([]sdkschema.SecurityGroup, len(sgs))
	for i := range sgs {
		items[i] = *securityGroupToAPI(sgs[i])
	}
	iterator := &sdknetwork.SecurityGroupIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: securitygroupdom.ProviderID,
			Resource: securitygroupdom.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator
}

// securityGroupFromAPI converts an SDK SecurityGroup to a SecurityGroup.
func securityGroupFromAPI(sdk sdkschema.SecurityGroup, id *SecurityGroupIdentity, region string) *securitygroupdom.SecurityGroup {
	sg := &securitygroupdom.SecurityGroup{
		Spec: securityGroupSpecFromAPI(sdk.Spec),
	}
	sg.Name = id.GetName()
	sg.ResourceVersion = id.GetVersion()
	sg.Provider = securitygroupdom.ProviderID
	sg.Tenant = id.GetTenant()
	sg.Workspace = id.GetWorkspace()
	sg.Region = region
	sg.Labels = sdk.Labels
	sg.Annotations = sdk.Annotations
	sg.Extensions = sdk.Extensions

	return sg
}

// newSecurityGroupWithIdentity returns a *securitygroupdom.SecurityGroup populated with identity fields from ir.
func newSecurityGroupWithIdentity(ir persistence.IdentifiableResource) *securitygroupdom.SecurityGroup {
	d := &securitygroupdom.SecurityGroup{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	return d
}

// securityGroupSpecToAPI converts a domain SecurityGroupSpec to an SDK SecurityGroupSpec.
func securityGroupSpecToAPI(spec securitygroupdom.SecurityGroupSpec) sdkschema.SecurityGroupSpec {
	out := sdkschema.SecurityGroupSpec{}
	for _, r := range spec.RuleRefs {
		out.RuleRefs = append(out.RuleRefs, commonfrontend.ReferenceToAPI(r))
	}
	for _, rule := range spec.Rules {
		out.Rules = append(out.Rules, securityGroupRuleSpecInlineToAPI(rule))
	}
	return out
}

// securityGroupSpecFromAPI converts an SDK SecurityGroupSpec to a domain SecurityGroupSpec.
func securityGroupSpecFromAPI(sdk sdkschema.SecurityGroupSpec) securitygroupdom.SecurityGroupSpec {
	spec := securitygroupdom.SecurityGroupSpec{}
	for _, r := range sdk.RuleRefs {
		spec.RuleRefs = append(spec.RuleRefs, commonfrontend.ReferenceFromAPI(r))
	}
	for _, rule := range sdk.Rules {
		spec.Rules = append(spec.Rules, securityGroupRuleSpecInlineFromAPI(rule))
	}
	return spec
}

// securityGroupRuleSpecInlineToAPI converts a domain SecurityGroup inline rule spec to its SDK representation.
func securityGroupRuleSpecInlineToAPI(spec securitygroupdom.SecurityGroupRuleSpec) sdkschema.SecurityGroupRuleSpec {
	out := sdkschema.SecurityGroupRuleSpec{
		Direction: sdkschema.SecurityGroupRuleSpecDirection(spec.Direction),
		Protocol:  sdkschema.SecurityGroupRuleSpecProtocol(spec.Protocol),
		Version:   commonfrontend.IPVersionToAPI(spec.Version),
	}
	if spec.Icmp != nil {
		out.Icmp = &sdkschema.IcmpConfig{Code: spec.Icmp.Code, Type: spec.Icmp.Type}
	}
	if spec.Ports != nil {
		out.Ports = &sdkschema.Ports{From: spec.Ports.From, To: spec.Ports.To, List: spec.Ports.List}
	}
	for _, r := range spec.SourceRef {
		out.SourceRef = append(out.SourceRef, commonfrontend.ReferenceToAPI(r))
	}
	return out
}

// securityGroupRuleSpecInlineFromAPI converts an SDK inline rule spec to the domain SecurityGroup representation.
func securityGroupRuleSpecInlineFromAPI(sdk sdkschema.SecurityGroupRuleSpec) securitygroupdom.SecurityGroupRuleSpec {
	spec := securitygroupdom.SecurityGroupRuleSpec{
		Direction: string(sdk.Direction),
		Protocol:  string(sdk.Protocol),
		Version:   commonfrontend.IPVersionFromAPI(sdk.Version),
	}
	if sdk.Icmp != nil {
		spec.Icmp = &securitygroupdom.IcmpConfig{Code: sdk.Icmp.Code, Type: sdk.Icmp.Type}
	}
	if sdk.Ports != nil {
		spec.Ports = &securitygroupdom.Ports{From: sdk.Ports.From, To: sdk.Ports.To, List: sdk.Ports.List}
	}
	for _, r := range sdk.SourceRef {
		spec.SourceRef = append(spec.SourceRef, commonfrontend.ReferenceFromAPI(r))
	}
	return spec
}
