// Package backend provides shared CR↔domain mapper helpers used by resource-specific backends.
package backend

import (
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
//
// A reference is stored and returned exactly as the client wrote it. Both representations
// the spec allows — the scope as its own fields ({tenant: "t", resource: "skus/s"}) and the
// scope spelled out in the path ({resource: "seca.network/v1/tenants/t/skus/s"}) — mean the
// same thing, and rewriting one into the other loses whichever the client chose, so a read
// no longer echoes the write. Anything that needs the pieces parses them at the point of
// use; see ParseReference.
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
func ReferenceFromCR(ref schemav1.Reference) domain.Reference {
	return domain.Reference{
		Provider:  ref.Provider,
		Region:    ref.Region,
		Resource:  ref.Resource,
		Tenant:    ref.Tenant,
		Workspace: ref.Workspace,
	}
}

// ReferenceToCR converts a domain.Reference to a generated schemav1.Reference, storing it
// verbatim. See ReferenceFromCR for why nothing is rewritten on the way in either.
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
func ReferenceToCR(ref domain.Reference) schemav1.Reference {
	return schemav1.Reference{
		Provider:  ref.Provider,
		Region:    ref.Region,
		Resource:  ref.Resource,
		Tenant:    ref.Tenant,
		Workspace: ref.Workspace,
	}
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
