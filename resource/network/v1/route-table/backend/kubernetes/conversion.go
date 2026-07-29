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
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

// RouteTableFromCR converts either a concrete *RouteTable or *unstructured.Unstructured
// into a *routetabledom.RouteTable.
func RouteTableFromCR(obj client.Object) (*routetabledom.RouteTable, error) {
	var cr RouteTable

	switch t := obj.(type) {
	case *RouteTable:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to RouteTable: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	routes := make([]routetabledom.RouteSpec, len(cr.Spec.Routes))
	for i, route := range cr.Spec.Routes {
		routes[i] = routetabledom.RouteSpec{
			DestinationCidrBlock: route.DestinationCidrBlock,
			TargetRef:            commonbackend.ReferenceFromCR(route.TargetRef),
		}
	}

	spec := routetabledom.RouteTableSpec{
		Routes: routes,
	}

	rt := &routetabledom.RouteTable{Spec: spec}
	rt.Name = cr.GetName()
	rt.ResourceVersion = cr.GetResourceVersion()
	rt.CreatedAt = cr.GetCreationTimestamp().Time
	rt.UpdatedAt = cr.GetCreationTimestamp().Time
	rt.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	rt.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	rt.Workspace = internalLabels[k8slabels.InternalWorkspaceLabel]
	rt.Network = internalLabels[k8slabels.InternalNetworkLabel]
	rt.Region = internalLabels[k8slabels.InternalRegionLabel]
	rt.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	rt.Annotations = cr.CommonData.Annotations
	rt.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		rt.DeletedAt = &ts.Time
	}

	rt.Status = &routetabledom.RouteTableStatus{}
	if cr.Status != nil {
		rt.Status.State = commonbackend.ResourceStateFromCR(cr.Status.State)
		rt.Status.Conditions = commonbackend.ConditionsFromCR(cr.Status.Conditions)

		routeStatuses := make([]routetabledom.RouteStatus, len(cr.Status.Routes))
		for i, rs := range cr.Status.Routes {
			routeStatuses[i] = routetabledom.RouteStatus{
				State:      commonbackend.ResourceStateFromCR(rs.State),
				Conditions: commonbackend.ConditionsFromCR(rs.Conditions),
			}
		}
		rt.Status.Routes = routeStatuses
	} else {
		rt.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return rt, nil
}

// RouteTableToCR converts a *routetabledom.RouteTable to a Kubernetes RouteTable CR.
func RouteTableToCR(rt *routetabledom.RouteTable) (client.Object, error) {
	if rt == nil {
		return nil, fmt.Errorf("route table is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(rt.Labels)
	crLabels[k8slabels.InternalTenantLabel] = rt.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = rt.Workspace
	crLabels[k8slabels.InternalNetworkLabel] = rt.Network
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(rt.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = rt.Region

	routes := make([]RouteSpec, len(rt.Spec.Routes))
	for i, route := range rt.Spec.Routes {
		routes[i] = RouteSpec{
			DestinationCidrBlock: route.DestinationCidrBlock,
			TargetRef:            commonbackend.ReferenceToCR(route.TargetRef),
		}
	}

	cr := &RouteTable{
		ObjectMeta: v1.ObjectMeta{
			Name:            rt.Name,
			Namespace:       k8sadapter.ComputeNetworkNamespace(rt),
			Labels:          crLabels,
			ResourceVersion: rt.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: rt.Annotations,
			Extensions:  rt.Extensions,
			Labels:      slices.Collect(maps.Keys(rt.Labels)),
		},
		Spec: RouteTableSpec{
			Routes: routes,
		},
	}
	cr.SetGroupVersionKind(RouteTableGVK)

	if rt.Status != nil && len(rt.Status.Conditions) > 0 {
		state := commonbackend.ResourceStateToCR(rt.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}

		routeStatuses := make([]RouteStatus, len(rt.Status.Routes))
		for i, rs := range rt.Status.Routes {
			rsState := commonbackend.ResourceStateToCR(rs.State)
			if rsState == nil {
				return nil, fmt.Errorf("failed to convert route status state to CR")
			}
			routeStatuses[i] = RouteStatus{
				Conditions: commonbackend.ConditionsToCR(rs.Conditions),
				State:      *rsState,
			}
		}

		cr.Status = &RouteTableStatus{
			Conditions: commonbackend.ConditionsToCR(rt.Status.Conditions),
			State:      *state,
			Routes:     routeStatuses,
		}
	}

	return cr, nil
}
