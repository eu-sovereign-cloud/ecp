package rest

import (
	"fmt"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

// SecurityGroupRuleIdentity carries identity for a single security group rule resource.
type SecurityGroupRuleIdentity struct {
	name            string
	tenant          string
	workspace       string
	resourceVersion string
}

func (sgr *SecurityGroupRuleIdentity) GetName() string      { return sgr.name }
func (sgr *SecurityGroupRuleIdentity) GetVersion() string   { return sgr.resourceVersion }
func (sgr *SecurityGroupRuleIdentity) GetTenant() string    { return sgr.tenant }
func (sgr *SecurityGroupRuleIdentity) GetWorkspace() string { return sgr.workspace }

var _ persistence.IdentifiableResource = (*SecurityGroupRuleIdentity)(nil)

// securityGroupRuleListParamsFromAPI converts SDK ListSecurityGroupRulesParams to resource.ListParams.
func securityGroupRuleListParamsFromAPI(params sdknetwork.ListSecurityGroupRulesParams, tenant, workspace string) resource.ListParams {
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

// securityGroupRuleToAPIWithVerb returns a func that converts a SecurityGroupRule to its SDK representation with the given verb.
func securityGroupRuleToAPIWithVerb(verb string) func(sgr *securitygroupruledom.SecurityGroupRule) *sdkschema.SecurityGroupRule {
	return func(sgr *securitygroupruledom.SecurityGroupRule) *sdkschema.SecurityGroupRule {
		sdk := securityGroupRuleToAPI(sgr)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

func securityGroupRuleToAPI(sgr *securitygroupruledom.SecurityGroupRule) *sdkschema.SecurityGroupRule {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(sgr.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	const kind = sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindSecurityGroupRule

	out := &sdkschema.SecurityGroupRule{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:     securitygroupruledom.Version,
			CreatedAt:      sgr.CreatedAt,
			LastModifiedAt: sgr.UpdatedAt,
			Kind:           kind,
			Name:           sgr.Name,
			Tenant:         sgr.Tenant,
			Workspace:      sgr.Workspace,
			Provider:       sgr.Provider,
			Region:         sgr.Region,
			Resource:       fmt.Sprintf(commondomain.RegionalResourceFormat, kind, sgr.Name),
			Ref: fmt.Sprintf(
				sgr.Provider+"/"+commondomain.RegionalWorkspaceScopedResourceFormat,
				sgr.Tenant, sgr.Workspace, kind, sgr.Name,
			),
			ResourceVersion: resourceVersion,
		},
		Labels:      sgr.Labels,
		Annotations: sgr.Annotations,
		Extensions:  sgr.Extensions,
		Spec:        securityGroupRuleSpecToAPI(sgr.Spec),
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}

	if sgr.Status != nil {
		out.Status = &sdkschema.SecurityGroupRuleStatus{
			Conditions: commonfrontend.ConditionsToAPI(sgr.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(sgr.Status.State),
		}
	}
	if sgr.DeletedAt != nil {
		out.Metadata.DeletedAt = sgr.DeletedAt
	}
	return out
}

// securityGroupRuleIteratorToAPI converts a list of SecurityGroupRule to an SDK SecurityGroupRuleIterator.
func securityGroupRuleIteratorToAPI(sgrs []*securitygroupruledom.SecurityGroupRule, nextSkipToken *string) *sdknetwork.SecurityGroupRuleIterator {
	items := make([]sdkschema.SecurityGroupRule, len(sgrs))
	for i := range sgrs {
		items[i] = *securityGroupRuleToAPI(sgrs[i])
	}
	iterator := &sdknetwork.SecurityGroupRuleIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: securitygroupruledom.ProviderID,
			Resource: securitygroupruledom.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator
}

// securityGroupRuleFromAPI converts an SDK SecurityGroupRule to a SecurityGroupRule.
func securityGroupRuleFromAPI(sdk sdkschema.SecurityGroupRule, id *SecurityGroupRuleIdentity, region string) *securitygroupruledom.SecurityGroupRule {
	sgr := &securitygroupruledom.SecurityGroupRule{
		Spec: securityGroupRuleSpecFromAPI(sdk.Spec),
	}
	sgr.Name = id.GetName()
	sgr.ResourceVersion = id.GetVersion()
	sgr.Provider = securitygroupruledom.ProviderID
	sgr.Tenant = id.GetTenant()
	sgr.Workspace = id.GetWorkspace()
	sgr.Region = region
	sgr.Labels = sdk.Labels
	sgr.Annotations = sdk.Annotations
	sgr.Extensions = sdk.Extensions

	return sgr
}

// newSecurityGroupRuleWithIdentity returns a *securitygroupruledom.SecurityGroupRule populated with identity fields from ir.
func newSecurityGroupRuleWithIdentity(ir persistence.IdentifiableResource) *securitygroupruledom.SecurityGroupRule {
	d := &securitygroupruledom.SecurityGroupRule{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	return d
}

// securityGroupRuleSpecToAPI converts a domain SecurityGroupRuleSpec to an SDK SecurityGroupRuleSpec.
func securityGroupRuleSpecToAPI(spec securitygroupruledom.SecurityGroupRuleSpec) sdkschema.SecurityGroupRuleSpec {
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

// securityGroupRuleSpecFromAPI converts an SDK SecurityGroupRuleSpec to a domain SecurityGroupRuleSpec.
func securityGroupRuleSpecFromAPI(sdk sdkschema.SecurityGroupRuleSpec) securitygroupruledom.SecurityGroupRuleSpec {
	spec := securitygroupruledom.SecurityGroupRuleSpec{
		Direction: string(sdk.Direction),
		Protocol:  string(sdk.Protocol),
		Version:   commonfrontend.IPVersionFromAPI(sdk.Version),
	}
	if sdk.Icmp != nil {
		spec.Icmp = &securitygroupruledom.IcmpConfig{Code: sdk.Icmp.Code, Type: sdk.Icmp.Type}
	}
	if sdk.Ports != nil {
		spec.Ports = &securitygroupruledom.Ports{From: sdk.Ports.From, To: sdk.Ports.To, List: sdk.Ports.List}
	}
	for _, r := range sdk.SourceRef {
		spec.SourceRef = append(spec.SourceRef, commonfrontend.ReferenceFromAPI(r))
	}
	return spec
}
