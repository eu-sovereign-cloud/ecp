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
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"

	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// RoleAssignmentFromCR converts either a concrete *RoleAssignment or *unstructured.Unstructured
// into a *radom.RoleAssignment.
func RoleAssignmentFromCR(obj client.Object) (*radom.RoleAssignment, error) {
	var cr RoleAssignment

	switch t := obj.(type) {
	case *RoleAssignment:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to RoleAssignment: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	ra := &radom.RoleAssignment{
		Spec: radom.RoleAssignmentSpec{
			Subs:   cr.Spec.Subs,
			Scopes: scopesFromCR(cr.Spec.Scopes),
			Roles:  cr.Spec.Roles,
		},
	}
	ra.Name = cr.GetName()
	ra.ResourceVersion = cr.GetResourceVersion()
	ra.CreatedAt = cr.GetCreationTimestamp().Time
	ra.UpdatedAt = cr.GetCreationTimestamp().Time
	ra.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	ra.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	ra.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	ra.Annotations = cr.CommonData.Annotations
	ra.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		ra.DeletedAt = &ts.Time
	}

	ra.Status = &radom.RoleAssignmentStatus{}
	if cr.Status != nil {
		ra.Status.State = commonbackend.ResourceStateFromCR(cr.Status.State)
		ra.Status.Conditions = commonbackend.ConditionsFromCR(cr.Status.Conditions)
	} else {
		ra.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return ra, nil
}

// RoleAssignmentToCR converts a *radom.RoleAssignment to a Kubernetes RoleAssignment CR.
func RoleAssignmentToCR(ra *radom.RoleAssignment) (client.Object, error) {
	if ra == nil {
		return nil, fmt.Errorf("role assignment is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(ra.Labels)
	crLabels[k8slabels.InternalTenantLabel] = ra.Tenant
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(ra.Provider, "/", "_")

	cr := &RoleAssignment{
		ObjectMeta: v1.ObjectMeta{
			Name:            ra.Name,
			Namespace:       k8sadapter.ComputeNamespace(tenantOnlyScope(ra.Tenant)),
			Labels:          crLabels,
			ResourceVersion: ra.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: ra.Annotations,
			Extensions:  ra.Extensions,
			Labels:      slices.Collect(maps.Keys(ra.Labels)),
		},
		Spec: RoleAssignmentSpec{
			Subs:   ra.Spec.Subs,
			Scopes: scopesToCR(ra.Spec.Scopes),
			Roles:  ra.Spec.Roles,
		},
	}
	cr.SetGroupVersionKind(RoleAssignmentGVK)

	if ra.Status != nil && len(ra.Status.Conditions) > 0 {
		state := commonbackend.ResourceStateToCR(ra.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}
		cr.Status = &RoleAssignmentStatus{
			Conditions: commonbackend.ConditionsToCR(ra.Status.Conditions),
			State:      *state,
		}
	}

	return cr, nil
}

// scopesFromCR converts CR role assignment scopes into their domain representation.
func scopesFromCR(scopes []RoleAssignmentScope) []radom.RoleAssignmentScope {
	if scopes == nil {
		return nil
	}
	out := make([]radom.RoleAssignmentScope, len(scopes))
	for i, s := range scopes {
		out[i] = radom.RoleAssignmentScope{
			Tenants:    s.Tenants,
			Regions:    s.Regions,
			Workspaces: s.Workspaces,
		}
	}
	return out
}

// scopesToCR converts domain role assignment scopes into their CR representation.
func scopesToCR(scopes []radom.RoleAssignmentScope) []RoleAssignmentScope {
	if scopes == nil {
		return nil
	}
	out := make([]RoleAssignmentScope, len(scopes))
	for i, s := range scopes {
		out[i] = RoleAssignmentScope{
			Tenants:    s.Tenants,
			Regions:    s.Regions,
			Workspaces: s.Workspaces,
		}
	}
	return out
}

// tenantOnlyScope returns a scope with only the tenant set.
// RoleAssignment CRs live in the tenant namespace (role assignments are tenant-scoped).
func tenantOnlyScope(tenant string) *kernelresource.Scope {
	return &kernelresource.Scope{Tenant: tenant}
}
