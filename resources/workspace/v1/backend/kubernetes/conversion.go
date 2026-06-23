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

	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	convert "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/convert"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/labels"
	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"

	commonbackend "github.com/eu-sovereign-cloud/ecp/resources/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/workspace/v1"
)

// MapCRToWorkspaceDomain converts either a concrete *Workspace or *unstructured.Unstructured
// into a *wsdom.Workspace.
func MapCRToWorkspaceDomain(obj client.Object) (*wsdom.Workspace, error) {
	var cr Workspace

	switch t := obj.(type) {
	case *Workspace:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Workspace: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	spec := make(wsdom.WorkspaceSpec, len(cr.Spec))
	for k, v := range cr.Spec {
		spec[k] = convert.StringToInterface(v)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	wd := &wsdom.Workspace{
		Spec: spec,
	}
	wd.Name = cr.GetName()
	wd.ResourceVersion = cr.GetResourceVersion()
	wd.CreatedAt = cr.GetCreationTimestamp().Time
	wd.UpdatedAt = cr.GetCreationTimestamp().Time
	wd.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	wd.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	wd.Region = internalLabels[k8slabels.InternalRegionLabel]
	wd.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	wd.Annotations = cr.CommonData.Annotations
	wd.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		wd.DeletedAt = &ts.Time
	}

	wd.Status = &wsdom.WorkspaceStatus{}
	if cr.Status != nil {
		wd.Status = &wsdom.WorkspaceStatus{
			ResourceCount: cr.Status.ResourceCount,
		}
		wd.Status.State = commondomain.ResourceStateDomain(cr.Status.State)
		wd.Status.Conditions = commonbackend.MapCRToStatusConditionDomains(cr.Status.Conditions)
	} else {
		wd.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return wd, nil
}

// MapWorkspaceDomainToCR converts a *wsdom.Workspace to a Kubernetes Workspace CR.
func MapWorkspaceDomainToCR(d *wsdom.Workspace) (client.Object, error) {
	if d == nil {
		return nil, fmt.Errorf("domain workspace is nil")
	}

	spec := make(WorkspaceSpec, len(d.Spec))
	for k, v := range d.Spec {
		spec[k] = convert.InterfaceToString(v)
	}

	crLabels := k8slabels.OriginalToKeyed(d.Labels)
	crLabels[k8slabels.InternalTenantLabel] = d.Tenant
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(d.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = d.Region

	cr := &Workspace{
		ObjectMeta: v1.ObjectMeta{
			Name:            d.Name,
			Namespace:       k8sadapter.ComputeNamespace(tenantOnlyScope(d.Tenant)),
			Labels:          crLabels,
			ResourceVersion: d.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: d.Annotations,
			Extensions:  d.Extensions,
			Labels:      slices.Collect(maps.Keys(d.Labels)),
		},
		Spec: spec,
	}
	cr.SetGroupVersionKind(WorkspaceGVK)

	if d.Status != nil && (len(d.Status.Conditions) > 0 || d.Status.ResourceCount != nil) {
		state := commonbackend.MapResourceStateDomainToCR(d.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to map resource state domain to CR")
		}
		cr.Status = &WorkspaceStatus{
			State:         *state,
			Conditions:    commonbackend.MapStatusConditionDomainsToCR(d.Status.Conditions),
			ResourceCount: d.Status.ResourceCount,
		}
	}

	return cr, nil
}

// tenantOnlyScope returns a scope with only the tenant set.
// Workspace CRs live in the tenant namespace (not in the workspace namespace).
func tenantOnlyScope(tenant string) *kernelresource.Scope {
	return &kernelresource.Scope{Tenant: tenant}
}
