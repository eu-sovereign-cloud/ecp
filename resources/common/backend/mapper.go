// Package backend provides shared CR↔domain mapper helpers used by resource-specific backends.
package backend

import (
	"fmt"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"

	"github.com/eu-sovereign-cloud/ecp/resources/common/domain"
)

// MapStatusConditionDomainToCR maps a domain.StatusConditionDomain to a genv1.StatusCondition.
func MapStatusConditionDomainToCR(domainStatusCondition domain.StatusConditionDomain) genv1.StatusCondition {
	var state genv1.ResourceState
	if mappedState := MapResourceStateDomainToCR(domainStatusCondition.State); mappedState != nil {
		state = *mappedState
	}

	return genv1.StatusCondition{
		Type:             domainStatusCondition.Type,
		State:            state,
		LastTransitionAt: v1.NewTime(domainStatusCondition.LastTransitionAt),
		Reason:           domainStatusCondition.Reason,
		Message:          domainStatusCondition.Message,
		Occurrences:      domainStatusCondition.Occurrences,
	}
}

// MapStatusConditionDomainsToCR maps a slice of domain.StatusConditionDomain to a slice of genv1.StatusCondition.
func MapStatusConditionDomainsToCR(domainStatusConditions []domain.StatusConditionDomain) []genv1.StatusCondition {
	conditions := make([]genv1.StatusCondition, len(domainStatusConditions))
	for i, cond := range domainStatusConditions {
		conditions[i] = MapStatusConditionDomainToCR(cond)
	}
	return conditions
}

// MapResourceStateDomainToCR maps domain.ResourceStateDomain to genv1.ResourceState.
func MapResourceStateDomainToCR(domainResourceState domain.ResourceStateDomain) *genv1.ResourceState {
	var state genv1.ResourceState
	switch domainResourceState {
	case domain.ResourceStatePending:
		state = genv1.ResourceStatePending
	case domain.ResourceStateCreating:
		state = genv1.ResourceStateCreating
	case domain.ResourceStateActive:
		state = genv1.ResourceStateActive
	case domain.ResourceStateUpdating:
		state = genv1.ResourceStateUpdating
	case domain.ResourceStateDeleting:
		state = genv1.ResourceStateDeleting
	case domain.ResourceStateError:
		state = genv1.ResourceStateError
	default:
		return nil
	}
	return &state
}

// MapCRToStatusConditionDomain maps a genv1.StatusCondition to a domain.StatusConditionDomain.
func MapCRToStatusConditionDomain(crStatusCondition genv1.StatusCondition) domain.StatusConditionDomain {
	return domain.StatusConditionDomain{
		Type:             crStatusCondition.Type,
		State:            MapCRToResourceStateDomain(crStatusCondition.State),
		LastTransitionAt: crStatusCondition.LastTransitionAt.Time,
		Reason:           crStatusCondition.Reason,
		Message:          crStatusCondition.Message,
		Occurrences:      crStatusCondition.Occurrences,
	}
}

// MapCRToStatusConditionDomains maps a slice of genv1.StatusCondition to a slice of domain.StatusConditionDomain.
func MapCRToStatusConditionDomains(crStatusConditions []genv1.StatusCondition) []domain.StatusConditionDomain {
	conditions := make([]domain.StatusConditionDomain, len(crStatusConditions))
	for i, cond := range crStatusConditions {
		conditions[i] = MapCRToStatusConditionDomain(cond)
	}
	return conditions
}

// MapCRToResourceStateDomain maps genv1.ResourceState to domain.ResourceStateDomain.
func MapCRToResourceStateDomain(crResourceState genv1.ResourceState) domain.ResourceStateDomain {
	var state domain.ResourceStateDomain
	switch crResourceState {
	case genv1.ResourceStatePending:
		state = domain.ResourceStatePending
	case genv1.ResourceStateCreating:
		state = domain.ResourceStateCreating
	case genv1.ResourceStateActive:
		state = domain.ResourceStateActive
	case genv1.ResourceStateUpdating:
		state = domain.ResourceStateUpdating
	case genv1.ResourceStateDeleting:
		state = domain.ResourceStateDeleting
	case genv1.ResourceStateError:
		state = domain.ResourceStateError
	default:
		state = ""
	}
	return state
}

// MapCRToReferenceDomain converts a generated genv1.Reference to a domain.ReferenceDomain.
// Tenant and Workspace are embedded into the Resource path so the domain always
// carries a fully-qualified resource string (e.g. "seca.storage/v1/tenants/t/skus/s").
func MapCRToReferenceDomain(ref genv1.Reference) domain.ReferenceDomain {
	resource := ref.Resource
	if ref.Tenant != "" || ref.Workspace != "" {
		resource = embedScopeInResource(resource, ref.Tenant, ref.Workspace)
	}
	return domain.ReferenceDomain{
		Provider: ref.Provider,
		Region:   ref.Region,
		Resource: resource,
	}
}

// MapReferenceDomainToCR converts a domain.ReferenceDomain to a generated genv1.Reference.
// It parses the Resource path to extract embedded segments (providers, regions, tenants, workspaces)
// and sets the corresponding fields. Extracted segments are stripped from the Resource path.
// If a segment is not in the path, it falls back to the domain value.
func MapReferenceDomainToCR(ref domain.ReferenceDomain) genv1.Reference {
	resource := ref.Resource
	result := genv1.Reference{}

	// Populate each field from the Resource path only when the explicit domain field
	// is not already set. This makes the function idempotent: on the first call the
	// embedded path segments are extracted; on subsequent calls (after a round-trip
	// through the CR) the explicit fields are already populated and path extraction
	// is skipped, leaving the Resource unchanged.
	if ref.Provider == "" {
		if provider, remaining := extractAndStripSegment(resource, "providers/"); provider != "" {
			result.Provider = provider
			resource = remaining
		}
	} else {
		result.Provider = ref.Provider
	}

	if ref.Region == "" {
		if region, remaining := extractAndStripSegment(resource, "regions/"); region != "" {
			result.Region = region
			resource = remaining
		}
	} else {
		result.Region = ref.Region
	}

	if ref.Tenant == "" {
		if tenant, remaining := extractAndStripSegment(resource, "tenants/"); tenant != "" {
			result.Tenant = tenant
			resource = remaining
		}
	} else {
		result.Tenant = ref.Tenant
	}

	if ref.Workspace == "" {
		if workspace, remaining := extractAndStripSegment(resource, "workspaces/"); workspace != "" {
			result.Workspace = workspace
			resource = remaining
		}
	} else {
		result.Workspace = ref.Workspace
	}

	result.Resource = resource
	return result
}

// embedScopeInResource inserts tenants/{tenant} and workspaces/{workspace} segments
// into the resource path, just before the resource type/name suffix.
// e.g. "seca.storage/v1/skus/fast-local" with tenant "seca" becomes
// "seca.storage/v1/tenants/seca/skus/fast-local".
func embedScopeInResource(resource, tenant, workspace string) string {
	// Find the resource type/name (last two path segments)
	lastSlash := strings.LastIndex(resource, "/")
	if lastSlash < 0 {
		return resource
	}
	secondLastSlash := strings.LastIndex(resource[:lastSlash], "/")

	var prefix, suffix string
	if secondLastSlash >= 0 {
		prefix = resource[:secondLastSlash]
		suffix = resource[secondLastSlash+1:]
	} else {
		prefix = ""
		suffix = resource
	}

	var scopePath string
	switch {
	case tenant != "" && workspace != "":
		scopePath = fmt.Sprintf("tenants/%s/workspaces/%s", tenant, workspace)
	case tenant != "":
		scopePath = fmt.Sprintf("tenants/%s", tenant)
	case workspace != "":
		scopePath = fmt.Sprintf("workspaces/%s", workspace)
	}

	if prefix != "" {
		return prefix + "/" + scopePath + "/" + suffix
	}
	return scopePath + "/" + suffix
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
