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
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

// PublicIpFromCR converts either a concrete *PublicIP or *unstructured.Unstructured into a *publicipdom.PublicIp.
func PublicIpFromCR(obj client.Object) (*publicipdom.PublicIp, error) {
	var cr PublicIP

	switch t := obj.(type) {
	case *PublicIP:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert unstructured to PublicIP: %w", err))
		}
	default:
		return nil, kernel.NewError(kernel.KindInternal, fmt.Errorf("unsupported object type %T", obj))
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	version, err := commonbackend.IPVersionFromCR(cr.Spec.Version)
	if err != nil {
		return nil, fmt.Errorf("public ip %s: %w", cr.Name, err)
	}

	spec := publicipdom.PublicIpSpec{
		Address: cr.Spec.Address,
		Version: version,
	}

	p := &publicipdom.PublicIp{Spec: spec}
	p.Name = cr.GetName()
	p.ResourceVersion = cr.GetResourceVersion()
	p.CreatedAt = cr.GetCreationTimestamp().Time
	p.UpdatedAt = cr.GetCreationTimestamp().Time
	p.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	p.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	p.Workspace = internalLabels[k8slabels.InternalWorkspaceLabel]
	p.Region = internalLabels[k8slabels.InternalRegionLabel]
	p.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	p.Annotations = cr.CommonData.Annotations
	p.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		p.DeletedAt = &ts.Time
	}

	p.Status = &publicipdom.PublicIpStatus{}
	if cr.Status != nil {
		status, err := commonbackend.StatusFromCR(cr.Status.State, cr.Status.Conditions)
		if err != nil {
			return nil, fmt.Errorf("public ip %s: %w", cr.Name, err)
		}
		p.Status.Status = status
		p.Status.IpAddress = cr.Status.IpAddress
		if cr.Status.AttachedTo != nil {
			ref := commonbackend.ReferenceFromCR(*cr.Status.AttachedTo)
			p.Status.AttachedTo = &ref
		}
	} else {
		p.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return p, nil
}

// PublicIpToCR converts a *publicipdom.PublicIp to a Kubernetes PublicIP CR.
func PublicIpToCR(p *publicipdom.PublicIp) (client.Object, error) {
	if p == nil {
		return nil, kernel.NewError(kernel.KindInternal, fmt.Errorf("public ip is nil"))
	}

	crLabels := k8slabels.OriginalToKeyed(p.Labels)
	crLabels[k8slabels.InternalTenantLabel] = p.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = p.Workspace
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(p.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = p.Region

	cr := &PublicIP{
		ObjectMeta: v1.ObjectMeta{
			Name:            p.Name,
			Namespace:       k8sadapter.ComputeNamespace(p),
			Labels:          crLabels,
			ResourceVersion: p.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: p.Annotations,
			Extensions:  p.Extensions,
			Labels:      slices.Sorted(maps.Keys(p.Labels)),
		},
		Spec: PublicIpSpec{
			Address: p.Spec.Address,
			Version: commonbackend.IPVersionToCR(p.Spec.Version),
		},
	}
	cr.SetGroupVersionKind(PublicIPGVK)

	if p.Status != nil && len(p.Status.Conditions) > 0 {
		state, conds, err := commonbackend.StatusToCR(p.Status.Status)
		if err != nil {
			return nil, fmt.Errorf("public ip %s: %w", p.Name, err)
		}
		cr.Status = &PublicIpStatus{
			Conditions: conds,
			State:      state,
			IpAddress:  p.Status.IpAddress,
		}
		if p.Status.AttachedTo != nil {
			ref := commonbackend.ReferenceToCR(*p.Status.AttachedTo)
			cr.Status.AttachedTo = &ref
		}
	}

	return cr, nil
}

// Converter is the CR<->domain conversion pair for PublicIp.
var Converter = k8sadapter.TwoWayConverter[*publicipdom.PublicIp]{
	FromCR: PublicIpFromCR,
	ToCR:   PublicIpToCR,
}
