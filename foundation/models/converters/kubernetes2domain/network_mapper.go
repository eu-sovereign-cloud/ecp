package kubernetes2domain

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/common"
	networksv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network/networks/v1"
	netowrkskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network/skus/v1"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/generated/types"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/adapters/kubernetes"

	model "github.com/eu-sovereign-cloud/ecp/foundation/models/domain"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/labels"
)

// MapCRToNetworkDomain converts either concrete *networksv1.Network or unstructured.Unstructured into a *regional.NetworkDomain.
func MapCRToNetworkDomain(obj client.Object) (*regional.NetworkDomain, error) {
	var cr networksv1.Network

	switch t := obj.(type) {
	case *networksv1.Network:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Network: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	spec := regional.NetworkSpecDomain{
		Cidr:          mapCRToCidrDomain(cr.Spec.Cidr),
		SkuRef:        mapCRToReferenceDomain(cr.Spec.SkuRef),
		RouteTableRef: mapCRToReferenceDomain(cr.Spec.RouteTableRef),
	}
	for _, c := range cr.Spec.AdditionalCidrs {
		spec.AdditionalCidrs = append(spec.AdditionalCidrs, mapCRToCidrDomain(c))
	}

	crLabels := cr.GetLabels()
	internalLabels := labels.GetInternalLabels(crLabels)
	keyedLabels := labels.GetKeyedLabels(crLabels)

	meta := regional.Metadata{
		CommonMetadata: model.CommonMetadata{
			Name:            cr.GetName(),
			ResourceVersion: cr.GetResourceVersion(),
			CreatedAt:       cr.GetCreationTimestamp().Time,
			UpdatedAt:       cr.GetCreationTimestamp().Time,
			Provider:        strings.ReplaceAll(internalLabels[labels.InternalProviderLabel], "_", "/"),
		},
		Scope: scope.Scope{
			Tenant:    internalLabels[labels.InternalTenantLabel],
			Workspace: internalLabels[labels.InternalWorkspaceLabel],
		},
		Region:      internalLabels[labels.InternalRegionLabel],
		Labels:      labels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels),
		Annotations: cr.CommonData.Annotations,
		Extensions:  cr.CommonData.Extensions,
	}
	if ts := cr.GetDeletionTimestamp(); ts != nil {
		meta.DeletedAt = &ts.Time
	}

	var status = &regional.NetworkStatusDomain{}
	if cr.Status != nil {
		status = &regional.NetworkStatusDomain{
			StatusDomain: regional.StatusDomain{
				State:      regional.ResourceStateDomain(cr.Status.State),
				Conditions: mapCRToStatusConditionDomains(cr.Status.Conditions),
			},
		}
	} else {
		status.PushCondition(regional.DefaultPendingCondition)
	}

	return &regional.NetworkDomain{
		Metadata: meta,
		Spec:     spec,
		Status:   status,
	}, nil
}

// MapNetworkDomainToCR converts a NetworkDomain to a Kubernetes Network CR.
func MapNetworkDomainToCR(domain *regional.NetworkDomain) (client.Object, error) {
	crLabels := labels.OriginalToKeyed(domain.Labels)
	crLabels[labels.InternalTenantLabel] = domain.Tenant
	crLabels[labels.InternalWorkspaceLabel] = domain.Workspace
	crLabels[labels.InternalProviderLabel] = strings.ReplaceAll(domain.Provider, "/", "_")
	crLabels[labels.InternalRegionLabel] = domain.Region

	additionalCidrs := make([]genv1.Cidr, len(domain.Spec.AdditionalCidrs))
	for i, c := range domain.Spec.AdditionalCidrs {
		additionalCidrs[i] = mapCidrDomainToCR(c)
	}

	cr := &networksv1.Network{
		ObjectMeta: v1.ObjectMeta{
			Name:            domain.Name,
			Namespace:       kubernetes.ComputeNamespace(domain),
			Labels:          crLabels,
			ResourceVersion: domain.ResourceVersion,
		},
		CommonData: common.CommonData{
			Annotations: domain.Annotations,
			Extensions:  domain.Extensions,
			Labels:      slices.Collect(maps.Keys(domain.Labels)),
		},
		Spec: genv1.NetworkSpec{
			Cidr:            mapCidrDomainToCR(domain.Spec.Cidr),
			AdditionalCidrs: additionalCidrs,
			SkuRef:          mapReferenceDomainToCR(domain.Spec.SkuRef),
			RouteTableRef:   mapReferenceDomainToCR(domain.Spec.RouteTableRef),
		},
	}
	cr.SetGroupVersionKind(networksv1.NetworkGVK)

	if domain.Status != nil && len(domain.Status.Conditions) > 0 {
		state := mapResourceStateDomainToCR(domain.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}
		cr.Status = &genv1.NetworkStatus{
			Conditions: mapStatusConditionDomainsToCR(domain.Status.Conditions),
			State:      *state,
		}
	}

	return cr, nil
}

// MapCRToNetworkSKUDomain converts a concrete *networkskuv1.SKU into a NetworkSKUDomain.
func MapCRToNetworkSKUDomain(cr netowrkskuv1.SKU) *regional.NetworkSKUDomain {
	return &regional.NetworkSKUDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: cr.GetName(),
			},
		},
		Spec: regional.NetworkSKUSpecDomain{
			Bandwidth: cr.Spec.Bandwidth,
			Packets:   cr.Spec.Packets,
		},
	}
}

// mapCRToCidrDomain converts a genv1.Cidr to a regional.CidrDomain.
func mapCRToCidrDomain(cr genv1.Cidr) regional.CidrDomain {
	return regional.CidrDomain{
		IPv4: cr.Ipv4,
		IPv6: cr.Ipv6,
	}
}

// mapCidrDomainToCR converts a regional.CidrDomain to a genv1.Cidr.
func mapCidrDomainToCR(domain regional.CidrDomain) genv1.Cidr {
	return genv1.Cidr{
		Ipv4: domain.IPv4,
		Ipv6: domain.IPv6,
	}
}
