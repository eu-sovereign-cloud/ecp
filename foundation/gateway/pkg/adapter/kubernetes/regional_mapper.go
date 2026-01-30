package kubernetes

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/common"
	netowrkskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/network/skus/v1"
	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/block-storages/v1"
	storageskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/skus/v1"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/convert"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

func MapCRToNetworkSKUDomain(cr netowrkskuv1.SKU) *regional.NetworkSKUDomain {
	return &regional.NetworkSKUDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: cr.GetName(),
			},
		},
		Spec: regional.NetworkSKUSpec{
			Bandwidth: cr.Spec.Bandwidth,
			Packets:   cr.Spec.Packets,
		},
	}
}

// MapCRToStorageSKUDomain converts either concrete *storageskuv1.StorageSKU or unstructured.Unstructured into a StorageSKUDomain.
func MapCRToStorageSKUDomain(obj client.Object) (*regional.StorageSKUDomain, error) {
	var cr storageskuv1.SKU

	switch t := obj.(type) {
	case *storageskuv1.SKU:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to StorageSKU: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := labels.GetInternalLabels(crLabels)
	meta := regional.Metadata{
		Labels: labels.GetCSPLabels(crLabels),
		CommonMetadata: model.CommonMetadata{
			Name:            cr.GetName(),
			ResourceVersion: cr.GetResourceVersion(),
			Provider:        internalLabels[labels.InternalProviderLabel],
			CreatedAt:       cr.GetCreationTimestamp().Time,
			UpdatedAt:       cr.GetCreationTimestamp().Time,
		},
		Region: internalLabels[labels.InternalRegionLabel],
		Scope: scope.Scope{
			Tenant: internalLabels[labels.InternalTenantLabel],
		},
	}
	if ts := cr.GetDeletionTimestamp(); ts != nil {
		meta.DeletedAt = &ts.Time
	}

	return &regional.StorageSKUDomain{
		Metadata: meta,
		Spec: regional.StorageSKUSpec{
			Iops:          int64(cr.Spec.Iops),
			MinVolumeSize: int64(cr.Spec.MinVolumeSize),
			Type:          string(cr.Spec.Type),
		},
	}, nil
}

// MapCRToWorkspaceDomain converts either concrete *workspacev1.Workspace or unstructured.Unstructured into a *regional.WorkspaceDomain.
func MapCRToWorkspaceDomain(obj client.Object) (*regional.WorkspaceDomain, error) {
	var cr workspacev1.Workspace

	switch t := obj.(type) {
	case *workspacev1.Workspace:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Workspace: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	spec := make(map[string]interface{}, len(cr.Spec))
	for k, v := range cr.Spec {
		spec[k] = convert.StringToInterface(v)
	}

	crLabels := cr.GetLabels()
	internalLabels := labels.GetInternalLabels(crLabels)
	keyedLabels := labels.GetKeyedLabels(crLabels)
	// NOTE: Do we expect CSP labels on resources created by a user? If so, they'll need to be added as well.

	meta := regional.Metadata{
		CommonMetadata: model.CommonMetadata{
			Name:            cr.GetName(),
			ResourceVersion: cr.GetResourceVersion(),
			CreatedAt:       cr.GetCreationTimestamp().Time,
			Provider:        internalLabels[labels.InternalProviderLabel],
		},
		Scope: scope.Scope{
			Tenant: internalLabels[labels.InternalTenantLabel],
		},
		Region:      internalLabels[labels.InternalRegionLabel],
		Labels:      labels.KeyedToOriginal(keyedLabels, cr.RegionalCommonData.Labels),
		Annotations: cr.RegionalCommonData.Annotations,
		Extensions:  cr.RegionalCommonData.Extensions,
	}
	if ts := cr.GetDeletionTimestamp(); ts != nil {
		meta.DeletedAt = &ts.Time
	}

	var resourceState *regional.ResourceStateDomain
	var status *regional.WorkspaceStatusDomain
	if cr.Status != nil {
		if cr.Status.State != nil {
			rs := mapCRToResourceStateDomain(*cr.Status.State)
			resourceState = &rs
		}
		status = &regional.WorkspaceStatusDomain{
			StatusDomain: regional.StatusDomain{
				State:      resourceState,
				Conditions: mapCRToStatusConditionDomains(cr.Status.Conditions),
			},
			ResourceCount: cr.Status.ResourceCount,
		}
	}

	return &regional.WorkspaceDomain{
		Metadata: meta,
		Spec:     spec,
		Status:   status,
	}, nil
}

// MapWorkspaceDomainToCR maps a WorkspaceDomain to a Workspace CR.
func MapWorkspaceDomainToCR(domain *regional.WorkspaceDomain) (client.Object, error) {
	if domain == nil {
		return nil, fmt.Errorf("domain workspace is nil")
	}

	spec := make(map[string]string, len(domain.Spec))
	for k, v := range domain.Spec {
		spec[k] = convert.InterfaceToString(v)
	}

	crLabels := labels.OriginalToKeyed(domain.Labels)
	crLabels[labels.InternalTenantLabel] = domain.Tenant
	cr := &workspacev1.Workspace{
		ObjectMeta: v1.ObjectMeta{
			Name:            domain.Name,
			Namespace:       ComputeNamespace(domain),
			Labels:          crLabels,
			ResourceVersion: domain.ResourceVersion,
		},
		RegionalCommonData: common.RegionalCommonData{
			Annotations: domain.Annotations,
			Extensions:  domain.Extensions,
			Labels:      slices.Collect(maps.Keys(domain.Labels)),
		},
		Spec: spec,
	}
	cr.SetGroupVersionKind(workspacev1.WorkspaceGVK)

	if domain.Status != nil && (domain.Status.State != nil || len(domain.Status.Conditions) > 0 || domain.Status.ResourceCount != nil) {
		cr.Status = &genv1.WorkspaceStatus{
			State:         mapResourceStateDomainToCR(domain.Status.State),
			Conditions:    mapStatusConditionDomainsToCR(domain.Status.Conditions),
			ResourceCount: domain.Status.ResourceCount,
		}
	}

	return cr, nil
}

// mapStatusConditionDomainToCR maps a regional.StatusConditionDomain to a types.StatusCondition.
func mapStatusConditionDomainToCR(domainStatusCondition regional.StatusConditionDomain) genv1.StatusCondition {
	var state genv1.ResourceState
	if mappedState := mapResourceStateDomainToCR(&domainStatusCondition.State); mappedState != nil {
		state = *mappedState
	}

	return genv1.StatusCondition{
		Type:             ptr.To(domainStatusCondition.Type),
		State:            state,
		LastTransitionAt: v1.NewTime(domainStatusCondition.LastTransitionAt),
		Reason:           ptr.To(domainStatusCondition.Reason),
		Message:          ptr.To(domainStatusCondition.Message),
	}
}

// mapStatusConditionDomainsToCR maps a slice of regional.StatusConditionDomain to a slice of types.StatusCondition.
func mapStatusConditionDomainsToCR(domainStatusConditions []regional.StatusConditionDomain) []genv1.StatusCondition {
	conditions := make([]genv1.StatusCondition, len(domainStatusConditions))
	for i, cond := range domainStatusConditions {
		conditions[i] = mapStatusConditionDomainToCR(cond)
	}
	return conditions
}

// mapResourceStateDomainToCR maps regional.ResourceStateDomain to types.ResourceState.
func mapResourceStateDomainToCR(domainResourceState *regional.ResourceStateDomain) *genv1.ResourceState {
	if domainResourceState == nil {
		return nil
	}
	var state genv1.ResourceState
	switch *domainResourceState {
	case regional.ResourceStatePending:
		state = genv1.ResourceStatePending
	case regional.ResourceStateCreating:
		state = genv1.ResourceStateCreating
	case regional.ResourceStateActive:
		state = genv1.ResourceStateActive
	case regional.ResourceStateUpdating:
		state = genv1.ResourceStateUpdating
	case regional.ResourceStateDeleting:
		state = genv1.ResourceStateDeleting
	case regional.ResourceStateSuspended:
		state = genv1.ResourceStateSuspended
	case regional.ResourceStateError:
		state = genv1.ResourceStateError
	default:
		return nil
	}
	return &state
}

// mapCRToStatusConditionDomain maps a types.StatusCondition to a regional.StatusConditionDomain.
func mapCRToStatusConditionDomain(crStatusCondition genv1.StatusCondition) regional.StatusConditionDomain {
	return regional.StatusConditionDomain{
		Type:             ptr.Deref(crStatusCondition.Type, ""),
		State:            mapCRToResourceStateDomain(crStatusCondition.State),
		LastTransitionAt: crStatusCondition.LastTransitionAt.Time,
		Reason:           ptr.Deref(crStatusCondition.Reason, ""),
		Message:          ptr.Deref(crStatusCondition.Message, ""),
	}
}

// mapCRToStatusConditionDomains maps a slice of types.StatusCondition to a slice of regional.StatusConditionDomain.
func mapCRToStatusConditionDomains(crStatusConditions []genv1.StatusCondition) []regional.StatusConditionDomain {
	conditions := make([]regional.StatusConditionDomain, len(crStatusConditions))
	for i, cond := range crStatusConditions {
		conditions[i] = mapCRToStatusConditionDomain(cond)
	}
	return conditions
}

// mapCRToResourceStateDomain maps types.ResourceState to regional.ResourceStateDomain.
func mapCRToResourceStateDomain(crResourceState genv1.ResourceState) regional.ResourceStateDomain {
	var state regional.ResourceStateDomain
	switch crResourceState {
	case genv1.ResourceStatePending:
		state = regional.ResourceStatePending
	case genv1.ResourceStateCreating:
		state = regional.ResourceStateCreating
	case genv1.ResourceStateActive:
		state = regional.ResourceStateActive
	case genv1.ResourceStateUpdating:
		state = regional.ResourceStateUpdating
	case genv1.ResourceStateDeleting:
		state = regional.ResourceStateDeleting
	case genv1.ResourceStateSuspended:
		state = regional.ResourceStateSuspended
	case genv1.ResourceStateError:
		state = regional.ResourceStateError
	default:
		state = ""
	}
	return state
}

// MapCRToBlockStorageDomain converts either concrete *blockstoragev1.BlockStorage or unstructured.Unstructured into a BlockStorageDomain.
func MapCRToBlockStorageDomain(obj client.Object) (*regional.BlockStorageDomain, error) {
	var cr blockstoragev1.BlockStorage

	switch t := obj.(type) {
	case *blockstoragev1.BlockStorage:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to BlockStorage: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := labels.GetInternalLabels(crLabels)
	meta := regional.Metadata{
		Labels: labels.GetCSPLabels(crLabels),
		CommonMetadata: model.CommonMetadata{
			Name:            cr.GetName(),
			ResourceVersion: cr.GetResourceVersion(),
			Provider:        internalLabels[labels.InternalProviderLabel],
			CreatedAt:       cr.GetCreationTimestamp().Time,
			UpdatedAt:       cr.GetCreationTimestamp().Time,
		},
		Region: internalLabels[labels.InternalRegionLabel],
		Scope: scope.Scope{
			Tenant:    internalLabels[labels.InternalTenantLabel],
			Workspace: internalLabels[labels.InternalWorkspaceLabel],
		},
	}
	if ts := cr.GetDeletionTimestamp(); ts != nil {
		meta.DeletedAt = &ts.Time
	}

	domain := &regional.BlockStorageDomain{
		Metadata: meta,
		Spec: regional.BlockStorageSpec{
			SizeGB: cr.Spec.SizeGB,
			SkuRef: mapCRReferenceObjectToDomain(cr.Spec.SkuRef),
		},
	}

	if cr.Spec.SourceImageRef != nil {
		ref := mapCRReferenceObjectToDomain(*cr.Spec.SourceImageRef)
		domain.Spec.SourceImageRef = &ref
	}

	if cr.Status == nil {
		return domain, nil
	}

	domain.Status = &regional.BlockStorageStatus{
		SizeGB:     cr.Status.SizeGB,
		Conditions: mapCRToStatusConditionDomains(cr.Status.Conditions),
	}

	if cr.Status.AttachedTo != nil {
		ref := mapCRReferenceObjectToDomain(*cr.Status.AttachedTo)
		domain.Status.AttachedTo = &ref
	}

	if cr.Status.State != nil {
		state := regional.ResourceStateDomain(*cr.Status.State)
		domain.Status.State = &state
	}

	return domain, nil
}

// mapCRReferenceObjectToDomain converts a generated types.ReferenceObject to a domain ReferenceObject.
func mapCRReferenceObjectToDomain(ref genv1.ReferenceObject) regional.ReferenceObject {
	return regional.ReferenceObject{
		Provider:  ptr.Deref(ref.Provider, ""),
		Region:    ptr.Deref(ref.Region, ""),
		Resource:  ref.Resource,
		Tenant:    ptr.Deref(ref.Tenant, ""),
		Workspace: ptr.Deref(ref.Workspace, ""),
	}
}

// MapBlockStorageDomainToCR converts a BlockStorageDomain to a Kubernetes BlockStorage CR.
func MapBlockStorageDomainToCR(domain *regional.BlockStorageDomain) (client.Object, error) {

	cr := &blockstoragev1.BlockStorage{
		ObjectMeta: v1.ObjectMeta{
			Name:            domain.Name,
			Namespace:       ComputeNamespace(domain),
			ResourceVersion: domain.ResourceVersion,
		},
		Spec: genv1.BlockStorageSpec{
			SizeGB: domain.Spec.SizeGB,
			SkuRef: mapDomainReferenceObjectToCR(domain.Spec.SkuRef),
		},
	}
	cr.SetGroupVersionKind(blockstoragev1.BlockStorageGVK)

	// Merge CSP labels with internal labels
	allLabels := labels.OriginalToKeyed(domain.Labels)
	allLabels[labels.InternalTenantLabel] = domain.Tenant
	allLabels[labels.InternalWorkspaceLabel] = domain.Workspace
	if domain.Region != "" {
		allLabels[labels.InternalRegionLabel] = domain.Region
	}
	if domain.Provider != "" {
		allLabels[labels.InternalProviderLabel] = domain.Provider
	}
	cr.SetLabels(allLabels)

	if domain.Spec.SourceImageRef != nil {
		ref := mapDomainReferenceObjectToCR(*domain.Spec.SourceImageRef)
		cr.Spec.SourceImageRef = &ref
	}

	if domain.Status != nil && (domain.Status.State != nil || len(domain.Status.Conditions) > 0) {
		cr.Status = &genv1.BlockStorageStatus{
			SizeGB:     domain.Status.SizeGB,
			Conditions: mapStatusConditionDomainsToCR(domain.Status.Conditions),
			State:      mapResourceStateDomainToCR(domain.Status.State),
		}
		if domain.Status.AttachedTo != nil {
			ref := mapDomainReferenceObjectToCR(*domain.Status.AttachedTo)
			cr.Status.AttachedTo = &ref
		}
	}

	return cr, nil
}

// mapDomainReferenceObjectToCR converts a domain ReferenceObject to a generated types ReferenceObject.
// It parses the Resource path to extract embedded segments (providers, regions, tenants, workspaces)
// and sets the corresponding fields. Extracted segments are stripped from the Resource path.
// If a segment is not in the path, it falls back to the domain value.
func mapDomainReferenceObjectToCR(ref regional.ReferenceObject) genv1.ReferenceObject {
	resource := ref.Resource
	result := genv1.ReferenceObject{}

	// Extract values from Resource path or fall back to domain values
	if provider, remaining := extractAndStripSegment(resource, "providers/"); provider != "" {
		result.Provider = ptr.To(provider)
		resource = remaining
	} else if ref.Provider != "" {
		result.Provider = ptr.To(ref.Provider)
	}

	if region, remaining := extractAndStripSegment(resource, "regions/"); region != "" {
		result.Region = ptr.To(region)
		resource = remaining
	} else if ref.Region != "" {
		result.Region = ptr.To(ref.Region)
	}

	if tenant, remaining := extractAndStripSegment(resource, "tenants/"); tenant != "" {
		result.Tenant = ptr.To(tenant)
		resource = remaining
	} else if ref.Tenant != "" {
		result.Tenant = ptr.To(ref.Tenant)
	}

	if workspace, remaining := extractAndStripSegment(resource, "workspaces/"); workspace != "" {
		result.Workspace = ptr.To(workspace)
		resource = remaining
	} else if ref.Workspace != "" {
		result.Workspace = ptr.To(ref.Workspace)
	}

	result.Resource = resource
	return result
}

// extractAndStripSegment extracts the value following a segment prefix in a resource path
// and returns the remaining path with the segment removed.
// For example, extractAndStripSegment("workspaces/ws-1/block-storages/my-storage", "workspaces/")
// returns ("ws-1", "block-storages/my-storage").
// Returns empty strings if the segment is not found.
func extractAndStripSegment(resource, segment string) (value, remaining string) {
	var startIdx int
	var prefixLen int

	if strings.HasPrefix(resource, segment) {
		startIdx = len(segment)
		prefixLen = 0
	} else if idx := strings.Index(resource, "/"+segment); idx >= 0 {
		startIdx = idx + 1 + len(segment)
		prefixLen = idx
	} else {
		return "", ""
	}

	// Find the end of the value (next "/" or end of string)
	endIdx := strings.Index(resource[startIdx:], "/")
	if endIdx < 0 {
		// Segment is at the end, return the value and prefix as remaining
		value = resource[startIdx:]
		if prefixLen > 0 {
			remaining = resource[:prefixLen]
		}
		return value, remaining
	}

	value = resource[startIdx : startIdx+endIdx]
	// Build remaining: prefix + suffix after the segment
	suffix := resource[startIdx+endIdx+1:]
	if prefixLen > 0 {
		remaining = resource[:prefixLen] + "/" + suffix
	} else {
		remaining = suffix
	}
	return value, remaining
}
