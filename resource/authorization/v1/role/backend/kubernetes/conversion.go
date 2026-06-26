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

	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// RoleFromCR converts either a concrete *Role or *unstructured.Unstructured
// into a *roledom.Role.
func RoleFromCR(obj client.Object) (*roledom.Role, error) {
	var cr Role

	switch t := obj.(type) {
	case *Role:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Role: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	r := &roledom.Role{
		Spec: specFromCR(cr.Spec),
	}
	r.Name = cr.GetName()
	r.ResourceVersion = cr.GetResourceVersion()
	r.CreatedAt = cr.GetCreationTimestamp().Time
	r.UpdatedAt = cr.GetCreationTimestamp().Time
	r.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	r.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	r.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	r.Annotations = cr.CommonData.Annotations
	r.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		r.DeletedAt = &ts.Time
	}

	r.Status = &roledom.RoleStatus{}
	if cr.Status != nil {
		r.Status.State = commonbackend.ResourceStateFromCR(cr.Status.State)
		r.Status.Conditions = commonbackend.ConditionsFromCR(cr.Status.Conditions)
	} else {
		r.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return r, nil
}

// RoleToCR converts a *roledom.Role to a Kubernetes Role CR.
func RoleToCR(r *roledom.Role) (client.Object, error) {
	if r == nil {
		return nil, fmt.Errorf("role is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(r.Labels)
	crLabels[k8slabels.InternalTenantLabel] = r.Tenant
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(r.Provider, "/", "_")

	cr := &Role{
		ObjectMeta: v1.ObjectMeta{
			Name:            r.Name,
			Namespace:       k8sadapter.ComputeNamespace(tenantOnlyScope(r.Tenant)),
			Labels:          crLabels,
			ResourceVersion: r.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: r.Annotations,
			Extensions:  r.Extensions,
			Labels:      slices.Collect(maps.Keys(r.Labels)),
		},
		Spec: specToCR(r.Spec),
	}
	cr.SetGroupVersionKind(RoleGVK)

	if r.Status != nil && len(r.Status.Conditions) > 0 {
		state := commonbackend.ResourceStateToCR(r.Status.State)
		if state == nil {
			return nil, fmt.Errorf("role %s: failed to map resource state domain to CR", r.Name)
		}
		cr.Status = &RoleStatus{
			State:      *state,
			Conditions: commonbackend.ConditionsToCR(r.Status.Conditions),
		}
	}

	return cr, nil
}

// tenantOnlyScope returns a scope with only the tenant set.
// Role CRs live in the tenant namespace (global resource, tenant-scoped).
func tenantOnlyScope(tenant string) *kernelresource.Scope {
	return &kernelresource.Scope{Tenant: tenant}
}

// specFromCR converts a kubernetes RoleSpec to a domain RoleSpec.
func specFromCR(spec RoleSpec) roledom.RoleSpec {
	permissions := make([]roledom.Permission, len(spec.Permissions))
	for i, p := range spec.Permissions {
		resources := make([]string, len(p.Resources))
		copy(resources, p.Resources)
		verbs := make([]string, len(p.Verb))
		copy(verbs, p.Verb)
		permissions[i] = roledom.Permission{
			Provider:  p.Provider,
			Resources: resources,
			Verb:      verbs,
		}
	}
	return roledom.RoleSpec{Permissions: permissions}
}

// specToCR converts a domain RoleSpec to a kubernetes RoleSpec.
func specToCR(spec roledom.RoleSpec) RoleSpec {
	permissions := make([]Permission, len(spec.Permissions))
	for i, p := range spec.Permissions {
		resources := make([]string, len(p.Resources))
		copy(resources, p.Resources)
		verbs := make([]string, len(p.Verb))
		copy(verbs, p.Verb)
		permissions[i] = Permission{
			Provider:  p.Provider,
			Resources: resources,
			Verb:      verbs,
		}
	}
	return RoleSpec{Permissions: permissions}
}
