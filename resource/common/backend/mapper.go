// Package backend provides shared CR↔domain mapper helpers used by resource-specific backends.
package backend

import (
	"fmt"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// The mappers below are asymmetric about what they reject: inbound rejects, outbound trusts.
// See doc/CONVENTIONS.md §2 and §10 for why.

// StatusFromCR maps a CR's state and conditions to a domain.Status. It is everything a slice's
// XFromCR does with its CR status, so the slice wraps one error instead of two.
func StatusFromCR(state schemav1.ResourceState, conds []schemav1.StatusCondition) (domain.Status, error) {
	mappedState, err := ResourceStateFromCR(state)
	if err != nil {
		return domain.Status{}, err
	}

	mappedConds, err := ConditionsFromCR(conds)
	if err != nil {
		return domain.Status{}, err
	}

	return domain.Status{State: mappedState, Conditions: mappedConds}, nil
}

// StatusToCR maps a domain.Status to the state and conditions a CR status carries, the inverse
// of StatusFromCR.
func StatusToCR(s domain.Status) (schemav1.ResourceState, []schemav1.StatusCondition, error) {
	state, err := ResourceStateToCR(s.State)
	if err != nil {
		return "", nil, err
	}

	conds, err := ConditionsToCR(s.Conditions)
	if err != nil {
		return "", nil, err
	}

	return state, conds, nil
}

// StatusConditionToCR maps a domain.StatusCondition to a schemav1.StatusCondition.
func StatusConditionToCR(c domain.StatusCondition) (schemav1.StatusCondition, error) {
	state, err := ResourceStateToCR(c.State)
	if err != nil {
		return schemav1.StatusCondition{}, fmt.Errorf("condition %q: %w", c.Type, err)
	}

	return schemav1.StatusCondition{
		Type:             c.Type,
		State:            state,
		LastTransitionAt: v1.NewTime(c.LastTransitionAt),
		Reason:           c.Reason,
		Message:          c.Message,
		Occurrences:      c.Occurrences,
	}, nil
}

// ConditionsToCR maps a slice of domain.StatusCondition to a slice of schemav1.StatusCondition.
func ConditionsToCR(conds []domain.StatusCondition) ([]schemav1.StatusCondition, error) {
	conditions := make([]schemav1.StatusCondition, len(conds))
	for i, c := range conds {
		converted, err := StatusConditionToCR(c)
		if err != nil {
			return nil, err
		}
		conditions[i] = converted
	}
	return conditions, nil
}

// ResourceStateToCR maps domain.ResourceState to schemav1.ResourceState.
func ResourceStateToCR(state domain.ResourceState) (schemav1.ResourceState, error) {
	switch state {
	case domain.ResourceStatePending:
		return schemav1.ResourceStatePending, nil
	case domain.ResourceStateCreating:
		return schemav1.ResourceStateCreating, nil
	case domain.ResourceStateActive:
		return schemav1.ResourceStateActive, nil
	case domain.ResourceStateUpdating:
		return schemav1.ResourceStateUpdating, nil
	case domain.ResourceStateDeleting:
		return schemav1.ResourceStateDeleting, nil
	case domain.ResourceStateError:
		return schemav1.ResourceStateError, nil
	default:
		return "", kernel.NewError(kernel.KindValidation,
			fmt.Errorf("unmappable resource state %q", state),
			kernel.ErrorSource{Name: "status.state", Value: string(state)})
	}
}

// StatusConditionFromCR maps a schemav1.StatusCondition to a domain.StatusCondition.
func StatusConditionFromCR(c schemav1.StatusCondition) (domain.StatusCondition, error) {
	state, err := ResourceStateFromCR(c.State)
	if err != nil {
		return domain.StatusCondition{}, fmt.Errorf("condition %q: %w", c.Type, err)
	}

	return domain.StatusCondition{
		Type:             c.Type,
		State:            state,
		LastTransitionAt: c.LastTransitionAt.Time,
		Reason:           c.Reason,
		Message:          c.Message,
		Occurrences:      c.Occurrences,
	}, nil
}

// ConditionsFromCR maps a slice of schemav1.StatusCondition to a slice of domain.StatusCondition.
func ConditionsFromCR(conds []schemav1.StatusCondition) ([]domain.StatusCondition, error) {
	conditions := make([]domain.StatusCondition, len(conds))
	for i, c := range conds {
		converted, err := StatusConditionFromCR(c)
		if err != nil {
			return nil, err
		}
		conditions[i] = converted
	}
	return conditions, nil
}

// ResourceStateFromCR maps schemav1.ResourceState to domain.ResourceState.
func ResourceStateFromCR(state schemav1.ResourceState) (domain.ResourceState, error) {
	switch state {
	case "":
		return "", nil
	case schemav1.ResourceStatePending:
		return domain.ResourceStatePending, nil
	case schemav1.ResourceStateCreating:
		return domain.ResourceStateCreating, nil
	case schemav1.ResourceStateActive:
		return domain.ResourceStateActive, nil
	case schemav1.ResourceStateUpdating:
		return domain.ResourceStateUpdating, nil
	case schemav1.ResourceStateDeleting:
		return domain.ResourceStateDeleting, nil
	case schemav1.ResourceStateError:
		return domain.ResourceStateError, nil
	default:
		return "", kernel.NewError(kernel.KindValidation,
			fmt.Errorf("unknown resource state %q", state),
			kernel.ErrorSource{Name: "status.state", Value: string(state)})
	}
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
func IPVersionFromCR(v schemav1.IPVersion) (domain.IPVersion, error) {
	switch v {
	case "":
		return "", nil
	case schemav1.IPVersionIPv4:
		return domain.IPVersionIPv4, nil
	case schemav1.IPVersionIPv6:
		return domain.IPVersionIPv6, nil
	default:
		return "", kernel.NewError(kernel.KindValidation,
			fmt.Errorf("unknown ip version %q", v),
			kernel.ErrorSource{Name: "version", Value: string(v)})
	}
}

// ReferenceFromCR converts a generated schemav1.Reference to a domain.Reference.
//
// Do not normalize between the two representations the spec allows (see domain.Reference):
// rewriting one into the other means a read no longer echoes the write. Whatever needs the
// pieces parses them at the point of use; see ParseReference.
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

// extractSegment returns the value following a segment prefix in a resource path, matched at a
// path boundary. For example extractSegment("tenants/t-1/workspaces/ws-1/skus/s", "workspaces/")
// returns "ws-1". Returns "" if the segment is not present.
func extractSegment(resourcePath, segment string) string {
	rest, ok := strings.CutPrefix(resourcePath, segment)
	if !ok {
		if _, after, found := strings.Cut(resourcePath, "/"+segment); found {
			rest = after
		} else {
			return ""
		}
	}
	value, _, _ := strings.Cut(rest, "/")
	return value
}
