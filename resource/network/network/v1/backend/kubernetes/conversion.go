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
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/network/v1"
)

// MapCRToNetworkDomain converts either a concrete *Network or *unstructured.Unstructured
// into a *netdom.Network.
func MapCRToNetworkDomain(obj client.Object) (*netdom.Network, error) {
	var cr Network

	switch t := obj.(type) {
	case *Network:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Network: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	spec := netdom.NetworkSpec{
		Cidr:          mapCRToCidrDomain(cr.Spec.Cidr),
		SkuRef:        commonbackend.MapCRToReferenceDomain(cr.Spec.SkuRef),
		RouteTableRef: commonbackend.MapCRToReferenceDomain(cr.Spec.RouteTableRef),
	}
	for _, c := range cr.Spec.AdditionalCidrs {
		spec.AdditionalCidrs = append(spec.AdditionalCidrs, mapCRToCidrDomain(c))
	}

	nd := &netdom.Network{
		Spec: spec,
	}
	nd.Name = cr.GetName()
	nd.ResourceVersion = cr.GetResourceVersion()
	nd.CreatedAt = cr.GetCreationTimestamp().Time
	nd.UpdatedAt = cr.GetCreationTimestamp().Time
	nd.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	nd.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	nd.Workspace = internalLabels[k8slabels.InternalWorkspaceLabel]
	nd.Region = internalLabels[k8slabels.InternalRegionLabel]
	nd.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	nd.Annotations = cr.CommonData.Annotations
	nd.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		nd.DeletedAt = &ts.Time
	}

	nd.Status = &netdom.NetworkStatus{}
	if cr.Status != nil {
		nd.Status = &netdom.NetworkStatus{}
		nd.Status.State = commondomain.ResourceState(cr.Status.State)
		nd.Status.Conditions = commonbackend.MapCRToStatusConditionDomains(cr.Status.Conditions)
	} else {
		nd.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return nd, nil
}

// MapNetworkDomainToCR converts a *netdom.Network to a Kubernetes Network CR.
func MapNetworkDomainToCR(d *netdom.Network) (client.Object, error) {
	if d == nil {
		return nil, fmt.Errorf("domain network is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(d.Labels)
	crLabels[k8slabels.InternalTenantLabel] = d.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = d.Workspace
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(d.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = d.Region

	additionalCidrs := make([]schemav1.Cidr, len(d.Spec.AdditionalCidrs))
	for i, c := range d.Spec.AdditionalCidrs {
		additionalCidrs[i] = mapCidrDomainToCR(c)
	}

	cr := &Network{
		ObjectMeta: v1.ObjectMeta{
			Name:            d.Name,
			Namespace:       k8sadapter.ComputeNamespace(d),
			Labels:          crLabels,
			ResourceVersion: d.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: d.Annotations,
			Extensions:  d.Extensions,
			Labels:      slices.Collect(maps.Keys(d.Labels)),
		},
		Spec: NetworkSpec{
			Cidr:            mapCidrDomainToCR(d.Spec.Cidr),
			AdditionalCidrs: additionalCidrs,
			SkuRef:          commonbackend.MapReferenceDomainToCR(d.Spec.SkuRef),
			RouteTableRef:   commonbackend.MapReferenceDomainToCR(d.Spec.RouteTableRef),
		},
	}
	cr.SetGroupVersionKind(NetworkGVK)

	if d.Status != nil && len(d.Status.Conditions) > 0 {
		state := commonbackend.MapResourceStateDomainToCR(d.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}
		cr.Status = &NetworkStatus{
			Conditions: commonbackend.MapStatusConditionDomainsToCR(d.Status.Conditions),
			State:      *state,
		}
	}

	return cr, nil
}

func mapCRToCidrDomain(cr schemav1.Cidr) netdom.Cidr {
	return netdom.Cidr{
		IPv4: cr.Ipv4,
		IPv6: cr.Ipv6,
	}
}

func mapCidrDomainToCR(d netdom.Cidr) schemav1.Cidr {
	return schemav1.Cidr{
		Ipv4: d.IPv4,
		Ipv6: d.IPv6,
	}
}
