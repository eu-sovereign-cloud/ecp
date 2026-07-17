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
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

// SecurityGroupRuleFromCR converts either a concrete *SecurityGroupRule or *unstructured.Unstructured
// into a *securitygroupruledom.SecurityGroupRule.
func SecurityGroupRuleFromCR(obj client.Object) (*securitygroupruledom.SecurityGroupRule, error) {
	var cr SecurityGroupRule

	switch t := obj.(type) {
	case *SecurityGroupRule:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to SecurityGroupRule: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	spec := securityGroupRuleSpecFromCR(cr.Spec)

	sgr := &securitygroupruledom.SecurityGroupRule{Spec: spec}
	sgr.Name = cr.GetName()
	sgr.ResourceVersion = cr.GetResourceVersion()
	sgr.CreatedAt = cr.GetCreationTimestamp().Time
	sgr.UpdatedAt = cr.GetCreationTimestamp().Time
	sgr.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	sgr.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	sgr.Workspace = internalLabels[k8slabels.InternalWorkspaceLabel]
	sgr.Region = internalLabels[k8slabels.InternalRegionLabel]
	sgr.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	sgr.Annotations = cr.CommonData.Annotations
	sgr.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		sgr.DeletedAt = &ts.Time
	}

	sgr.Status = &securitygroupruledom.SecurityGroupRuleStatus{}
	if cr.Status != nil {
		sgr.Status.State = commonbackend.ResourceStateFromCR(cr.Status.State)
		sgr.Status.Conditions = commonbackend.ConditionsFromCR(cr.Status.Conditions)
	} else {
		sgr.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return sgr, nil
}

// SecurityGroupRuleToCR converts a *securitygroupruledom.SecurityGroupRule to a Kubernetes SecurityGroupRule CR.
func SecurityGroupRuleToCR(sgr *securitygroupruledom.SecurityGroupRule) (client.Object, error) {
	if sgr == nil {
		return nil, fmt.Errorf("security group rule is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(sgr.Labels)
	crLabels[k8slabels.InternalTenantLabel] = sgr.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = sgr.Workspace
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(sgr.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = sgr.Region

	cr := &SecurityGroupRule{
		ObjectMeta: v1.ObjectMeta{
			Name:            sgr.Name,
			Namespace:       k8sadapter.ComputeNamespace(sgr),
			Labels:          crLabels,
			ResourceVersion: sgr.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: sgr.Annotations,
			Extensions:  sgr.Extensions,
			Labels:      slices.Collect(maps.Keys(sgr.Labels)),
		},
		Spec: securityGroupRuleSpecToCR(sgr.Spec),
	}
	cr.SetGroupVersionKind(SecurityGroupRuleGVK)

	if sgr.Status != nil && len(sgr.Status.Conditions) > 0 {
		state := commonbackend.ResourceStateToCR(sgr.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}
		cr.Status = &SecurityGroupRuleStatus{
			Conditions: commonbackend.ConditionsToCR(sgr.Status.Conditions),
			State:      *state,
		}
	}

	return cr, nil
}

// securityGroupRuleSpecFromCR converts a CR SecurityGroupRuleSpec to a domain SecurityGroupRuleSpec.
func securityGroupRuleSpecFromCR(cr SecurityGroupRuleSpec) securitygroupruledom.SecurityGroupRuleSpec {
	spec := securitygroupruledom.SecurityGroupRuleSpec{
		Direction: string(cr.Direction),
		Protocol:  string(cr.Protocol),
		Version:   commonbackend.IPVersionFromCR(cr.Version),
	}
	if cr.Icmp != nil {
		spec.Icmp = &securitygroupruledom.IcmpConfig{Code: cr.Icmp.Code, Type: cr.Icmp.Type}
	}
	if cr.Ports != nil {
		spec.Ports = &securitygroupruledom.Ports{From: cr.Ports.From, To: cr.Ports.To, List: cr.Ports.List}
	}
	for _, r := range cr.SourceRef {
		spec.SourceRef = append(spec.SourceRef, commonbackend.ReferenceFromCR(r))
	}
	return spec
}

// securityGroupRuleSpecToCR converts a domain SecurityGroupRuleSpec to a CR SecurityGroupRuleSpec.
func securityGroupRuleSpecToCR(spec securitygroupruledom.SecurityGroupRuleSpec) SecurityGroupRuleSpec {
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
