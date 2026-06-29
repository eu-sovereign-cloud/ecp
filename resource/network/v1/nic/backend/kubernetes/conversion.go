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
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

// NicFromCR converts either a concrete *NIC or *unstructured.Unstructured into a *nicdom.Nic.
func NicFromCR(obj client.Object) (*nicdom.Nic, error) {
	var cr NIC

	switch t := obj.(type) {
	case *NIC:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to NIC: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	spec := nicdom.NicSpec{
		Addresses: cr.Spec.Addresses,
		SubnetRef: commonbackend.ReferenceFromCR(cr.Spec.SubnetRef),
	}
	if cr.Spec.SkuRef != nil {
		spec.SkuRef = commonbackend.ReferenceFromCR(*cr.Spec.SkuRef)
	}
	for _, r := range cr.Spec.PublicIpRefs {
		spec.PublicIpRefs = append(spec.PublicIpRefs, commonbackend.ReferenceFromCR(r))
	}
	for _, r := range cr.Spec.SecurityGroupRefs {
		spec.SecurityGroupRefs = append(spec.SecurityGroupRefs, commonbackend.ReferenceFromCR(r))
	}

	n := &nicdom.Nic{Spec: spec}
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

	n.Status = &nicdom.NicStatus{}
	if cr.Status != nil {
		n.Status.State = commonbackend.ResourceStateFromCR(cr.Status.State)
		n.Status.Conditions = commonbackend.ConditionsFromCR(cr.Status.Conditions)
		n.Status.MacAddress = cr.Status.MacAddress
		n.Status.Addresses = cr.Status.Addresses
		for _, r := range cr.Status.PublicIpRefs {
			n.Status.PublicIpRefs = append(n.Status.PublicIpRefs, commonbackend.ReferenceFromCR(r))
		}
	} else {
		n.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return n, nil
}

// NicToCR converts a *nicdom.Nic to a Kubernetes NIC CR.
func NicToCR(n *nicdom.Nic) (client.Object, error) {
	if n == nil {
		return nil, fmt.Errorf("nic is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(n.Labels)
	crLabels[k8slabels.InternalTenantLabel] = n.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = n.Workspace
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(n.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = n.Region

	publicIPRefs := make([]schemav1.Reference, len(n.Spec.PublicIpRefs))
	for i, r := range n.Spec.PublicIpRefs {
		publicIPRefs[i] = commonbackend.ReferenceToCR(r)
	}
	securityGroupRefs := make([]schemav1.Reference, len(n.Spec.SecurityGroupRefs))
	for i, r := range n.Spec.SecurityGroupRefs {
		securityGroupRefs[i] = commonbackend.ReferenceToCR(r)
	}

	spec := NicSpec{
		Addresses:         n.Spec.Addresses,
		SubnetRef:         commonbackend.ReferenceToCR(n.Spec.SubnetRef),
		PublicIpRefs:      publicIPRefs,
		SecurityGroupRefs: securityGroupRefs,
	}
	if n.Spec.SkuRef != (commondomain.Reference{}) {
		ref := commonbackend.ReferenceToCR(n.Spec.SkuRef)
		spec.SkuRef = &ref
	}

	cr := &NIC{
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
		Spec: spec,
	}
	cr.SetGroupVersionKind(NICGVK)

	if n.Status != nil && len(n.Status.Conditions) > 0 {
		state := commonbackend.ResourceStateToCR(n.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}
		statusPublicIpRefs := make([]schemav1.Reference, len(n.Status.PublicIpRefs))
		for i, r := range n.Status.PublicIpRefs {
			statusPublicIpRefs[i] = commonbackend.ReferenceToCR(r)
		}
		cr.Status = &NicStatus{
			Conditions:   commonbackend.ConditionsToCR(n.Status.Conditions),
			State:        *state,
			MacAddress:   n.Status.MacAddress,
			Addresses:    n.Status.Addresses,
			PublicIpRefs: statusPublicIpRefs,
		}
	}

	return cr, nil
}
