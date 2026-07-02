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
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

// NetworkFromCR converts either a concrete *Network or *unstructured.Unstructured
// into a *netdom.Network.
func NetworkFromCR(obj client.Object) (*netdom.Network, error) {
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
		CIDR:   cidrFromCR(cr.Spec.Cidr),
		SkuRef: commonbackend.ReferenceFromCR(cr.Spec.SkuRef),
	}
	for _, c := range cr.Spec.AdditionalCidrs {
		spec.AdditionalCIDRs = append(spec.AdditionalCIDRs, cidrFromCR(c))
	}

	n := &netdom.Network{
		Spec: spec,
	}
	n.Name = cr.GetName()
	n.ResourceVersion = cr.GetResourceVersion()
	n.CreatedAt = cr.GetCreationTimestamp().Time
	n.UpdatedAt = cr.GetCreationTimestamp().Time
	n.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	n.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	n.Workspace = internalLabels[k8slabels.InternalWorkspaceLabel]
	n.Region = internalLabels[k8slabels.InternalRegionLabel]
	n.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	n.Annotations = cr.CommonData.Annotations
	n.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		n.DeletedAt = &ts.Time
	}

	n.Status = &netdom.NetworkStatus{}
	if cr.Status != nil {
		n.Status = &netdom.NetworkStatus{}
		n.Status.State = commonbackend.ResourceStateFromCR(cr.Status.State)
		n.Status.Conditions = commonbackend.ConditionsFromCR(cr.Status.Conditions)
	} else {
		n.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return n, nil
}

// NetworkToCR converts a *netdom.Network to a Kubernetes Network CR.
func NetworkToCR(n *netdom.Network) (client.Object, error) {
	if n == nil {
		return nil, fmt.Errorf("network is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(n.Labels)
	crLabels[k8slabels.InternalTenantLabel] = n.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = n.Workspace
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(n.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = n.Region

	additionalCidrs := make([]schemav1.Cidr, len(n.Spec.AdditionalCIDRs))
	for i, c := range n.Spec.AdditionalCIDRs {
		additionalCidrs[i] = cidrToCR(c)
	}

	cr := &Network{
		ObjectMeta: v1.ObjectMeta{
			Name:            n.Name,
			Namespace:       k8sadapter.ComputeNamespace(n),
			Labels:          crLabels,
			ResourceVersion: n.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: n.Annotations,
			Extensions:  n.Extensions,
			Labels:      slices.Collect(maps.Keys(n.Labels)),
		},
		Spec: NetworkSpec{
			Cidr:            cidrToCR(n.Spec.CIDR),
			AdditionalCidrs: additionalCidrs,
			SkuRef:          commonbackend.ReferenceToCR(n.Spec.SkuRef),
		},
	}
	cr.SetGroupVersionKind(NetworkGVK)

	if n.Status != nil && len(n.Status.Conditions) > 0 {
		state := commonbackend.ResourceStateToCR(n.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}
		cr.Status = &NetworkStatus{
			Conditions: commonbackend.ConditionsToCR(n.Status.Conditions),
			State:      *state,
		}
	}

	return cr, nil
}

func cidrFromCR(cr schemav1.Cidr) netdom.CIDR {
	return netdom.CIDR{
		IPv4: cr.Ipv4,
		IPv6: cr.Ipv6,
	}
}

func cidrToCR(c netdom.CIDR) schemav1.Cidr {
	return schemav1.Cidr{
		Ipv4: c.IPv4,
		Ipv6: c.IPv6,
	}
}
