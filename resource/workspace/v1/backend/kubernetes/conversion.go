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
	convert "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/convert"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"

	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// WorkspaceFromCR converts either a concrete *Workspace or *unstructured.Unstructured
// into a *wsdom.Workspace.
func WorkspaceFromCR(obj client.Object) (*wsdom.Workspace, error) {
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

	ws := &wsdom.Workspace{
		Spec: spec,
	}
	ws.Name = cr.GetName()
	ws.ResourceVersion = cr.GetResourceVersion()
	ws.CreatedAt = cr.GetCreationTimestamp().Time
	ws.UpdatedAt = cr.GetCreationTimestamp().Time
	ws.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	ws.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	ws.Region = internalLabels[k8slabels.InternalRegionLabel]
	ws.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	ws.Annotations = cr.CommonData.Annotations
	ws.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		ws.DeletedAt = &ts.Time
	}

	ws.Status = &wsdom.WorkspaceStatus{}
	if cr.Status != nil {
		ws.Status = &wsdom.WorkspaceStatus{
			ResourceCount: cr.Status.ResourceCount,
		}
		ws.Status.State = commonbackend.ResourceStateFromCR(cr.Status.State)
		ws.Status.Conditions = commonbackend.ConditionsFromCR(cr.Status.Conditions)
	} else {
		ws.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return ws, nil
}

// WorkspaceToCR converts a *wsdom.Workspace to a Kubernetes Workspace CR.
func WorkspaceToCR(ws *wsdom.Workspace) (client.Object, error) {
	if ws == nil {
		return nil, fmt.Errorf("workspace is nil")
	}

	spec := make(WorkspaceSpec, len(ws.Spec))
	for k, v := range ws.Spec {
		spec[k] = convert.InterfaceToString(v)
	}

	crLabels := k8slabels.OriginalToKeyed(ws.Labels)
	crLabels[k8slabels.InternalTenantLabel] = ws.Tenant
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(ws.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = ws.Region

	cr := &Workspace{
		ObjectMeta: v1.ObjectMeta{
			Name:            ws.Name,
			Namespace:       k8sadapter.ComputeNamespace(tenantOnlyScope(ws.Tenant)),
			Labels:          crLabels,
			ResourceVersion: ws.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: ws.Annotations,
			Extensions:  ws.Extensions,
			Labels:      slices.Collect(maps.Keys(ws.Labels)),
		},
		Spec: spec,
	}
	cr.SetGroupVersionKind(WorkspaceGVK)

	if ws.Status != nil && (len(ws.Status.Conditions) > 0 || ws.Status.ResourceCount != nil) {
		state := commonbackend.ResourceStateToCR(ws.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to map resource state domain to CR")
		}
		cr.Status = &WorkspaceStatus{
			State:         *state,
			Conditions:    commonbackend.ConditionsToCR(ws.Status.Conditions),
			ResourceCount: ws.Status.ResourceCount,
		}
	}

	return cr, nil
}

// tenantOnlyScope returns a scope with only the tenant set.
// Workspace CRs live in the tenant namespace (not in the workspace namespace).
func tenantOnlyScope(tenant string) *kernelresource.Scope {
	return &kernelresource.Scope{Tenant: tenant}
}
