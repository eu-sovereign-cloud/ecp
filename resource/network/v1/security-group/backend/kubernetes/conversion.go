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
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"

	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

// SecurityGroupFromCR converts either a concrete *SecurityGroup or *unstructured.Unstructured
// into a *securitygroupdom.SecurityGroup.
func SecurityGroupFromCR(obj client.Object) (*securitygroupdom.SecurityGroup, error) {
	var cr SecurityGroup

	switch t := obj.(type) {
	case *SecurityGroup:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert unstructured to SecurityGroup: %w", err))
		}
	default:
		return nil, kernel.NewError(kernel.KindInternal, fmt.Errorf("unsupported object type %T", obj))
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	rules := make([]securitygroupdom.SecurityGroupRuleSpec, len(cr.Spec.Rules))
	for i, rule := range cr.Spec.Rules {
		ruleSpec, err := securityGroupRuleSpecFromCR(rule)
		if err != nil {
			return nil, fmt.Errorf("security group %s: %w", cr.Name, err)
		}
		rules[i] = ruleSpec
	}

	spec := securitygroupdom.SecurityGroupSpec{
		Rules: rules,
	}
	for _, r := range cr.Spec.RuleRefs {
		spec.RuleRefs = append(spec.RuleRefs, commonbackend.ReferenceFromCR(r))
	}

	sg := &securitygroupdom.SecurityGroup{Spec: spec}
	sg.Name = cr.GetName()
	sg.ResourceVersion = cr.GetResourceVersion()
	sg.CreatedAt = cr.GetCreationTimestamp().Time
	sg.UpdatedAt = cr.GetCreationTimestamp().Time
	sg.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	sg.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	sg.Workspace = internalLabels[k8slabels.InternalWorkspaceLabel]
	sg.Region = internalLabels[k8slabels.InternalRegionLabel]
	sg.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	sg.Annotations = cr.CommonData.Annotations
	sg.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		sg.DeletedAt = &ts.Time
	}

	sg.Status = &securitygroupdom.SecurityGroupStatus{}
	if cr.Status != nil {
		state, err := commonbackend.ResourceStateFromCR(cr.Status.State)
		if err != nil {
			return nil, fmt.Errorf("security group %s: %w", cr.Name, err)
		}
		conds, err := commonbackend.ConditionsFromCR(cr.Status.Conditions)
		if err != nil {
			return nil, fmt.Errorf("security group %s: %w", cr.Name, err)
		}
		sg.Status.State = state
		sg.Status.Conditions = conds

		ruleStatuses := make([]securitygroupdom.SecurityGroupRuleStatus, len(cr.Status.Rules))
		for i, rs := range cr.Status.Rules {
			rsState, err := commonbackend.ResourceStateFromCR(rs.State)
			if err != nil {
				return nil, fmt.Errorf("security group %s: %w", cr.Name, err)
			}
			rsConds, err := commonbackend.ConditionsFromCR(rs.Conditions)
			if err != nil {
				return nil, fmt.Errorf("security group %s: %w", cr.Name, err)
			}
			ruleStatuses[i] = securitygroupdom.SecurityGroupRuleStatus{
				State:      rsState,
				Conditions: rsConds,
			}
		}
		sg.Status.Rules = ruleStatuses
	} else {
		sg.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return sg, nil
}

// SecurityGroupToCR converts a *securitygroupdom.SecurityGroup to a Kubernetes SecurityGroup CR.
func SecurityGroupToCR(sg *securitygroupdom.SecurityGroup) (client.Object, error) {
	if sg == nil {
		return nil, kernel.NewError(kernel.KindInternal, fmt.Errorf("security group is nil"))
	}

	crLabels := k8slabels.OriginalToKeyed(sg.Labels)
	crLabels[k8slabels.InternalTenantLabel] = sg.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = sg.Workspace
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(sg.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = sg.Region

	rules := make([]SecurityGroupRuleSpec, len(sg.Spec.Rules))
	for i, rule := range sg.Spec.Rules {
		rules[i] = securityGroupRuleSpecToCR(rule)
	}
	ruleRefs := make([]schemav1.Reference, len(sg.Spec.RuleRefs))
	for i, r := range sg.Spec.RuleRefs {
		ruleRefs[i] = commonbackend.ReferenceToCR(r)
	}

	cr := &SecurityGroup{
		ObjectMeta: v1.ObjectMeta{
			Name:            sg.Name,
			Namespace:       k8sadapter.ComputeNamespace(sg),
			Labels:          crLabels,
			ResourceVersion: sg.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: sg.Annotations,
			Extensions:  sg.Extensions,
			Labels:      slices.Sorted(maps.Keys(sg.Labels)),
		},
		Spec: SecurityGroupSpec{
			RuleRefs: ruleRefs,
			Rules:    rules,
		},
	}
	cr.SetGroupVersionKind(SecurityGroupGVK)

	if sg.Status != nil && len(sg.Status.Conditions) > 0 {
		state, err := commonbackend.ResourceStateToCR(sg.Status.State)
		if err != nil {
			return nil, fmt.Errorf("security group %s: %w", sg.Name, err)
		}
		conds, err := commonbackend.ConditionsToCR(sg.Status.Conditions)
		if err != nil {
			return nil, fmt.Errorf("security group %s: %w", sg.Name, err)
		}

		ruleStatuses := make([]SecurityGroupRuleStatus, len(sg.Status.Rules))
		for i, rs := range sg.Status.Rules {
			rsState, err := commonbackend.ResourceStateToCR(rs.State)
			if err != nil {
				return nil, fmt.Errorf("security group %s: %w", sg.Name, err)
			}
			rsConds, err := commonbackend.ConditionsToCR(rs.Conditions)
			if err != nil {
				return nil, fmt.Errorf("security group %s: %w", sg.Name, err)
			}
			ruleStatuses[i] = SecurityGroupRuleStatus{
				Conditions: rsConds,
				State:      rsState,
			}
		}

		cr.Status = &SecurityGroupStatus{
			Conditions: conds,
			State:      state,
			Rules:      ruleStatuses,
		}
	}

	return cr, nil
}

// securityGroupRuleSpecFromCR converts a CR SecurityGroupRuleSpec to a domain SecurityGroupRuleSpec.
func securityGroupRuleSpecFromCR(cr SecurityGroupRuleSpec) (securitygroupdom.SecurityGroupRuleSpec, error) {
	version, err := commonbackend.IPVersionFromCR(cr.Version)
	if err != nil {
		return securitygroupdom.SecurityGroupRuleSpec{}, err
	}

	spec := securitygroupdom.SecurityGroupRuleSpec{
		Direction: string(cr.Direction),
		Protocol:  string(cr.Protocol),
		Version:   version,
	}
	if cr.Icmp != nil {
		spec.Icmp = &securitygroupdom.IcmpConfig{Code: cr.Icmp.Code, Type: cr.Icmp.Type}
	}
	if cr.Ports != nil {
		spec.Ports = &securitygroupdom.Ports{From: cr.Ports.From, To: cr.Ports.To, List: cr.Ports.List}
	}
	for _, r := range cr.SourceRef {
		spec.SourceRef = append(spec.SourceRef, commonbackend.ReferenceFromCR(r))
	}
	return spec, nil
}

// securityGroupRuleSpecToCR converts a domain SecurityGroupRuleSpec to a CR SecurityGroupRuleSpec.
func securityGroupRuleSpecToCR(spec securitygroupdom.SecurityGroupRuleSpec) SecurityGroupRuleSpec {
	cr := SecurityGroupRuleSpec{
		Direction: SecurityGroupRuleSpecDirection(spec.Direction),
		Protocol:  SecurityGroupRuleSpecProtocol(spec.Protocol),
		Version:   commonbackend.IPVersionToCR(spec.Version),
	}
	if spec.Icmp != nil {
		cr.Icmp = &IcmpConfig{Code: spec.Icmp.Code, Type: spec.Icmp.Type}
	}
	if spec.Ports != nil {
		cr.Ports = &Ports{From: spec.Ports.From, To: spec.Ports.To, List: spec.Ports.List}
	}
	for _, r := range spec.SourceRef {
		cr.SourceRef = append(cr.SourceRef, commonbackend.ReferenceToCR(r))
	}
	return cr
}

// Converter is the CR<->domain conversion pair for SecurityGroup, so a call site names one value
// instead of pairing the two directions by hand. See doc/CONVENTIONS.md §2.
var Converter = k8sadapter.TwoWayConverter[*securitygroupdom.SecurityGroup]{
	FromCR: SecurityGroupFromCR,
	ToCR:   SecurityGroupToCR,
}
