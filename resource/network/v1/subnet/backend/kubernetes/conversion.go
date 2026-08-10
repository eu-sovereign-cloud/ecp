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
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

// SubnetFromCR converts either a concrete *Subnet or *unstructured.Unstructured
// into a *subnetdom.Subnet.
func SubnetFromCR(obj client.Object) (*subnetdom.Subnet, error) {
	var cr Subnet

	switch t := obj.(type) {
	case *Subnet:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Subnet: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	spec := subnetdom.SubnetSpec{
		Cidr:          cidrFromCR(cr.Spec.Cidr),
		RouteTableRef: commonbackend.ReferenceFromCR(cr.Spec.RouteTableRef),
		Zone:          cr.Spec.Zone,
	}
	if cr.Spec.SkuRef != nil {
		spec.SkuRef = commonbackend.ReferenceFromCR(*cr.Spec.SkuRef)
	}

	s := &subnetdom.Subnet{Spec: spec}
	s.Name = cr.GetName()
	s.ResourceVersion = cr.GetResourceVersion()
	s.CreatedAt = cr.GetCreationTimestamp().Time
	s.UpdatedAt = cr.GetCreationTimestamp().Time
	s.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	s.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	s.Workspace = internalLabels[k8slabels.InternalWorkspaceLabel]
	s.Network = internalLabels[k8slabels.InternalNetworkLabel]
	s.Region = internalLabels[k8slabels.InternalRegionLabel]
	s.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	s.Annotations = cr.CommonData.Annotations
	s.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		s.DeletedAt = &ts.Time
	}

	s.Status = &subnetdom.SubnetStatus{}
	if cr.Status != nil {
		s.Status.State = commonbackend.ResourceStateFromCR(cr.Status.State)
		s.Status.Conditions = commonbackend.ConditionsFromCR(cr.Status.Conditions)
		if cr.Status.Cidr != nil {
			cidr := cidrFromCR(*cr.Status.Cidr)
			s.Status.Cidr = &cidr
		}
		if cr.Status.RouteTableRef != nil {
			ref := commonbackend.ReferenceFromCR(*cr.Status.RouteTableRef)
			s.Status.RouteTableRef = &ref
		}
	} else {
		s.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return s, nil
}

// SubnetToCR converts a *subnetdom.Subnet to a Kubernetes Subnet CR.
func SubnetToCR(s *subnetdom.Subnet) (client.Object, error) {
	if s == nil {
		return nil, fmt.Errorf("subnet is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(s.Labels)
	crLabels[k8slabels.InternalTenantLabel] = s.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = s.Workspace
	crLabels[k8slabels.InternalNetworkLabel] = s.Network
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(s.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = s.Region

	spec := SubnetSpec{
		Cidr:          cidrToCR(s.Spec.Cidr),
		RouteTableRef: commonbackend.ReferenceToCR(s.Spec.RouteTableRef),
		Zone:          s.Spec.Zone,
	}
	if s.Spec.SkuRef != (commondomain.Reference{}) {
		spec.SkuRef = new(commonbackend.ReferenceToCR(s.Spec.SkuRef))
	}

	cr := &Subnet{
		ObjectMeta: v1.ObjectMeta{
			Name:            s.Name,
			Namespace:       k8sadapter.ComputeNetworkNamespace(s),
			Labels:          crLabels,
			ResourceVersion: s.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: s.Annotations,
			Extensions:  s.Extensions,
			Labels:      slices.Sorted(maps.Keys(s.Labels)),
		},
		Spec: spec,
	}
	cr.SetGroupVersionKind(SubnetGVK)

	if s.Status != nil && len(s.Status.Conditions) > 0 {
		state := commonbackend.ResourceStateToCR(s.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}

		status := &SubnetStatus{
			Conditions: commonbackend.ConditionsToCR(s.Status.Conditions),
			State:      *state,
		}
		if s.Status.Cidr != nil {
			cidr := cidrToCR(*s.Status.Cidr)
			status.Cidr = &cidr
		}
		if s.Status.RouteTableRef != nil {
			ref := commonbackend.ReferenceToCR(*s.Status.RouteTableRef)
			status.RouteTableRef = &ref
		}
		cr.Status = status
	}

	return cr, nil
}

func cidrFromCR(cr schemav1.Cidr) subnetdom.CIDR {
	return subnetdom.CIDR{
		IPv4: cr.Ipv4,
		IPv6: cr.Ipv6,
	}
}

func cidrToCR(c subnetdom.CIDR) schemav1.Cidr {
	return schemav1.Cidr{
		Ipv4: c.IPv4,
		Ipv6: c.IPv6,
	}
}
