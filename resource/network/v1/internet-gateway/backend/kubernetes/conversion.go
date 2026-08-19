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
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// InternetGatewayFromCR converts either a concrete *InternetGateway or *unstructured.Unstructured
// into a *internetgatewaydom.InternetGateway.
func InternetGatewayFromCR(obj client.Object) (*internetgatewaydom.InternetGateway, error) {
	var cr InternetGateway

	switch t := obj.(type) {
	case *InternetGateway:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert unstructured to InternetGateway: %w", err))
		}
	default:
		return nil, kernel.NewError(kernel.KindInternal, fmt.Errorf("unsupported object type %T", obj))
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	spec := internetgatewaydom.InternetGatewaySpec{
		EgressOnly: cr.Spec.EgressOnly,
	}

	ig := &internetgatewaydom.InternetGateway{Spec: spec}
	ig.Name = cr.GetName()
	ig.ResourceVersion = cr.GetResourceVersion()
	ig.CreatedAt = cr.GetCreationTimestamp().Time
	ig.UpdatedAt = cr.GetCreationTimestamp().Time
	ig.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	ig.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	ig.Workspace = internalLabels[k8slabels.InternalWorkspaceLabel]
	ig.Region = internalLabels[k8slabels.InternalRegionLabel]
	ig.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	ig.Annotations = cr.CommonData.Annotations
	ig.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		ig.DeletedAt = &ts.Time
	}

	ig.Status = &internetgatewaydom.InternetGatewayStatus{}
	if cr.Status != nil {
		status, err := commonbackend.StatusFromCR(cr.Status.State, cr.Status.Conditions)
		if err != nil {
			return nil, fmt.Errorf("internet gateway %s: %w", cr.Name, err)
		}
		ig.Status.Status = status
	} else {
		ig.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return ig, nil
}

// InternetGatewayToCR converts a *internetgatewaydom.InternetGateway to a Kubernetes InternetGateway CR.
func InternetGatewayToCR(ig *internetgatewaydom.InternetGateway) (client.Object, error) {
	if ig == nil {
		return nil, kernel.NewError(kernel.KindInternal, fmt.Errorf("internet gateway is nil"))
	}

	crLabels := k8slabels.OriginalToKeyed(ig.Labels)
	crLabels[k8slabels.InternalTenantLabel] = ig.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = ig.Workspace
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(ig.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = ig.Region

	cr := &InternetGateway{
		ObjectMeta: v1.ObjectMeta{
			Name:            ig.Name,
			Namespace:       k8sadapter.ComputeNamespace(ig),
			Labels:          crLabels,
			ResourceVersion: ig.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: ig.Annotations,
			Extensions:  ig.Extensions,
			Labels:      slices.Sorted(maps.Keys(ig.Labels)),
		},
		Spec: InternetGatewaySpec{
			EgressOnly: ig.Spec.EgressOnly,
		},
	}
	cr.SetGroupVersionKind(InternetGatewayGVK)

	if ig.Status != nil && len(ig.Status.Conditions) > 0 {
		state, conds, err := commonbackend.StatusToCR(ig.Status.Status)
		if err != nil {
			return nil, fmt.Errorf("internet gateway %s: %w", ig.Name, err)
		}
		cr.Status = &InternetGatewayStatus{
			Conditions: conds,
			State:      state,
		}
	}

	return cr, nil
}

// Converter is the CR<->domain conversion pair for InternetGateway.
var Converter = k8sadapter.TwoWayConverter[*internetgatewaydom.InternetGateway]{
	FromCR: InternetGatewayFromCR,
	ToCR:   InternetGatewayToCR,
}
