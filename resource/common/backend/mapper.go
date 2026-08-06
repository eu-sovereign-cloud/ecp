// Package backend provides shared CR↔domain mapper helpers used by resource-specific backends.
package backend

import (
	"fmt"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// StatusConditionToCR maps a domain.StatusCondition to a schemav1.StatusCondition.
func StatusConditionToCR(c domain.StatusCondition) schemav1.StatusCondition {
	var state schemav1.ResourceState
	if mappedState := ResourceStateToCR(c.State); mappedState != nil {
		state = *mappedState
	}

	return schemav1.StatusCondition{
		Type:             c.Type,
		State:            state,
		LastTransitionAt: v1.NewTime(c.LastTransitionAt),
		Reason:           c.Reason,
		Message:          c.Message,
		Occurrences:      c.Occurrences,
	}
}

// ConditionsToCR maps a slice of domain.StatusCondition to a slice of schemav1.StatusCondition.
func ConditionsToCR(conds []domain.StatusCondition) []schemav1.StatusCondition {
	conditions := make([]schemav1.StatusCondition, len(conds))
	for i, cond := range conds {
		conditions[i] = StatusConditionToCR(cond)
	}
	return conditions
}

// ResourceStateToCR maps domain.ResourceState to schemav1.ResourceState.
func ResourceStateToCR(state domain.ResourceState) *schemav1.ResourceState {
	var out schemav1.ResourceState
	switch state {
	case domain.ResourceStatePending:
		out = schemav1.ResourceStatePending
	case domain.ResourceStateCreating:
		out = schemav1.ResourceStateCreating
	case domain.ResourceStateActive:
		out = schemav1.ResourceStateActive
	case domain.ResourceStateUpdating:
		out = schemav1.ResourceStateUpdating
	case domain.ResourceStateDeleting:
		out = schemav1.ResourceStateDeleting
	case domain.ResourceStateError:
		out = schemav1.ResourceStateError
	default:
		return nil
	}
	return &out
}

// StatusConditionFromCR maps a schemav1.StatusCondition to a domain.StatusCondition.
func StatusConditionFromCR(c schemav1.StatusCondition) domain.StatusCondition {
	return domain.StatusCondition{
		Type:             c.Type,
		State:            ResourceStateFromCR(c.State),
		LastTransitionAt: c.LastTransitionAt.Time,
		Reason:           c.Reason,
		Message:          c.Message,
		Occurrences:      c.Occurrences,
	}
}

// ConditionsFromCR maps a slice of schemav1.StatusCondition to a slice of domain.StatusCondition.
func ConditionsFromCR(conds []schemav1.StatusCondition) []domain.StatusCondition {
	conditions := make([]domain.StatusCondition, len(conds))
	for i, cond := range conds {
		conditions[i] = StatusConditionFromCR(cond)
	}
	return conditions
}

// ResourceStateFromCR maps schemav1.ResourceState to domain.ResourceState.
func ResourceStateFromCR(state schemav1.ResourceState) domain.ResourceState {
	var out domain.ResourceState
	switch state {
	case schemav1.ResourceStatePending:
		out = domain.ResourceStatePending
	case schemav1.ResourceStateCreating:
		out = domain.ResourceStateCreating
	case schemav1.ResourceStateActive:
		out = domain.ResourceStateActive
	case schemav1.ResourceStateUpdating:
		out = domain.ResourceStateUpdating
	case schemav1.ResourceStateDeleting:
		out = domain.ResourceStateDeleting
	case schemav1.ResourceStateError:
		out = domain.ResourceStateError
	default:
		out = ""
	}
	return out
}

// IPVersionToCR maps domain.IPVersion to schemav1.IPVersion.
func IPVersionToCR(v domain.IPVersion) schemav1.IPVersion {
	switch v {
	case domain.IPVersionIPv4:
		return schemav1.IPVersionIPv4
	case domain.IPVersionIPv6:
		return schemav1.IPVersionIPv6
	default:
		return ""
	}
}

// IPVersionFromCR maps schemav1.IPVersion to domain.IPVersion.
func IPVersionFromCR(v schemav1.IPVersion) domain.IPVersion {
	switch v {
	case schemav1.IPVersionIPv4:
		return domain.IPVersionIPv4
	case schemav1.IPVersionIPv6:
		return domain.IPVersionIPv6
	default:
		return ""
	}
}

// ReferenceFromCR converts a generated schemav1.Reference to a domain.Reference.
// Tenant and Workspace are embedded into the Resource path so the domain always
// carries a fully-qualified resource path string
// (e.g. "tenants/t/workspaces/w/block-storages/bs").
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
func ReferenceFromCR(ref schemav1.Reference) domain.Reference {
	resourcePath := ref.Resource
	if ref.Tenant != "" || ref.Workspace != "" {
		resourcePath = embedScopeInResource(resourcePath, ref.Tenant, ref.Workspace)
	}
	return domain.Reference{
		Provider: ref.Provider,
		Region:   ref.Region,
		Resource: resourcePath,
	}
}

// ReferenceToCR converts a domain.Reference to a generated schemav1.Reference.
// It parses the Resource path to extract embedded segments (tenants, workspaces;
// also legacy providers/regions) and sets the corresponding fields. Extracted
// segments are stripped from the Resource path, leaving {collection}/{name}
// (or nested network paths). If a segment is not in the path, it falls back to
// the domain value.
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
func ReferenceToCR(ref domain.Reference) schemav1.Reference {
	resourcePath := ref.Resource
	result := schemav1.Reference{}

	// Populate each field from the Resource path only when the explicit domain field
	// is not already set. This makes the function idempotent: on the first call the
	// embedded path segments are extracted; on subsequent calls (after a round-trip
	// through the CR) the explicit fields are already populated and path extraction
	// is skipped, leaving the Resource unchanged.
	if ref.Provider == "" {
		if provider, remaining := extractAndStripSegment(resourcePath, "providers/"); provider != "" {
			result.Provider = provider
			resourcePath = remaining
		}
	} else {
		result.Provider = ref.Provider
	}

	if ref.Region == "" {
		if region, remaining := extractAndStripSegment(resourcePath, "regions/"); region != "" {
			result.Region = region
			resourcePath = remaining
		}
	} else {
		result.Region = ref.Region
	}

	if ref.Tenant == "" {
		if tenant, remaining := extractAndStripSegment(resourcePath, "tenants/"); tenant != "" {
			result.Tenant = tenant
			resourcePath = remaining
		}
	} else {
		result.Tenant = ref.Tenant
	}

	if ref.Workspace == "" {
		if workspace, remaining := extractAndStripSegment(resourcePath, "workspaces/"); workspace != "" {
			result.Workspace = workspace
			resourcePath = remaining
		}
	} else {
		result.Workspace = ref.Workspace
	}

	result.Resource = resourcePath
	return result
}

// embedScopeInResource inserts the tenants/{tenant} and workspaces/{workspace} segments
// into the resource path, matching the URN segment order
// {provider}/{version}/tenants/{tenant}/workspaces/{workspace}/{resource}.
// e.g. "skus/fast-local" with tenant "t1" becomes "tenants/t1/skus/fast-local", and the
// nested "networks/n1/route-tables/rt1" becomes
// "tenants/t1/workspaces/w1/networks/n1/route-tables/rt1" — the scope precedes the whole
// path, never just its last two segments.
//
// A client holding only a resource's URN sends that URN as the reference path, so the path
// may open with the "{provider}/{version}" pair. That pair stays in front: the scope goes
// after it, never before it and never in the middle of the nested path, so the path handed
// back is byte-identical to the URN the client sent.
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
func embedScopeInResource(resourcePath, tenant, workspace string) string {
	var scopePath string
	switch {
	case tenant != "" && workspace != "":
		scopePath = fmt.Sprintf("tenants/%s/workspaces/%s", tenant, workspace)
	case tenant != "":
		scopePath = fmt.Sprintf("tenants/%s", tenant)
	case workspace != "":
		scopePath = fmt.Sprintf("workspaces/%s", workspace)
	}

	providerPath, rest := splitProviderPrefix(resourcePath)
	segments := make([]string, 0, 3)
	for _, s := range []string{providerPath, scopePath, rest} {
		if s != "" {
			segments = append(segments, s)
		}
	}
	return strings.Join(segments, "/")
}

// splitProviderPrefix splits a leading "{provider}/{version}" pair (e.g. "seca.network/v1")
// off a resource path, returning it and the remainder. A SECA provider id always carries a
// dot and is followed by a v-prefixed version, neither of which a collection segment
// ("networks", "route-tables", "tenants") ever has, so the pair is unambiguous. Returns an
// empty prefix when the path does not open with one.
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
func splitProviderPrefix(resourcePath string) (providerPath, rest string) {
	provider, afterProvider, found := strings.Cut(resourcePath, "/")
	if !found || !strings.Contains(provider, ".") {
		return "", resourcePath
	}
	version, afterVersion, found := strings.Cut(afterProvider, "/")
	if !found || len(version) < 2 || version[0] != 'v' || version[1] < '0' || version[1] > '9' {
		return "", resourcePath
	}
	return provider + "/" + version, afterVersion
}

// extractAndStripSegment extracts the value following a segment prefix in a resource path
// and returns the remaining path with the segment removed.
// For example, extractAndStripSegment("workspaces/ws-1/block-storages/my-storage", "workspaces/")
// returns ("ws-1", "block-storages/my-storage").
// Returns empty strings if the segment is not found.
func extractAndStripSegment(resourcePath, segment string) (value, remaining string) {
	var startIdx int
	var prefixLen int

	if strings.HasPrefix(resourcePath, segment) {
		startIdx = len(segment)
		prefixLen = 0
	} else if idx := strings.Index(resourcePath, "/"+segment); idx >= 0 {
		startIdx = idx + 1 + len(segment)
		prefixLen = idx
	} else {
		return "", ""
	}

	// Find the end of the value (next "/" or end of string)
	endIdx := strings.Index(resourcePath[startIdx:], "/")
	if endIdx < 0 {
		// Segment is at the end, return the value and prefix as remaining
		value = resourcePath[startIdx:]
		if prefixLen > 0 {
			remaining = resourcePath[:prefixLen]
		}
		return value, remaining
	}

	value = resourcePath[startIdx : startIdx+endIdx]
	// Build remaining: prefix + suffix after the segment
	suffix := resourcePath[startIdx+endIdx+1:]
	if prefixLen > 0 {
		remaining = resourcePath[:prefixLen] + "/" + suffix
	} else {
		remaining = suffix
	}
	return value, remaining
}
