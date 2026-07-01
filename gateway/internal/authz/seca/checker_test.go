package seca

import (
	"context"
	"errors"
	"testing"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
)

// Compile-time interface assertions for the test stubs.
var _ persistence.ReaderRepo[*roledom.Role] = (*stubRoleReader)(nil)
var _ persistence.ReaderRepo[*radom.RoleAssignment] = (*stubAssignmentReader)(nil)

// stubRoleReader is a minimal persistence.ReaderRepo[*roledom.Role] for tests.
// List returns the preconfigured roles or error; Load panics (not needed by Checker).
type stubRoleReader struct {
	roles []*roledom.Role
	err   error
}

func (s *stubRoleReader) List(_ context.Context, _ resource.ListParams, list *[]*roledom.Role) (*string, error) {
	if s.err != nil {
		return nil, s.err
	}
	*list = s.roles
	return nil, nil
}

func (s *stubRoleReader) Load(_ context.Context, _ **roledom.Role) error {
	panic("stubRoleReader.Load not implemented in test")
}

// stubAssignmentReader is the same for role assignments.
type stubAssignmentReader struct {
	assignments []*radom.RoleAssignment
	err         error
}

func (s *stubAssignmentReader) List(_ context.Context, _ resource.ListParams, list *[]*radom.RoleAssignment) (*string, error) {
	if s.err != nil {
		return nil, s.err
	}
	*list = s.assignments
	return nil, nil
}

func (s *stubAssignmentReader) Load(_ context.Context, _ **radom.RoleAssignment) error {
	panic("stubAssignmentReader.Load not implemented in test")
}

// TestChecker_Authorize covers the three outcomes of the reader-backed Checker:
// policy allow, policy deny, and technical/infrastructure failure.
//
// The core correctness of the RBAC algorithm (scope, resource, verb matching) is
// already exhaustively covered by TestEvaluate in evaluator_test.go. This test
// focuses on the Checker's adapter responsibility: translating Evaluate's bool and
// reader errors into the correct (Decision, error) pairs — in particular proving
// that a reader failure yields DecisionError (HTTP 500), not DecisionDenied (HTTP 403).
func TestChecker_Authorize(t *testing.T) {
	t.Parallel()

	// A minimal role + assignment that permits seca.compute/instances:list everywhere.
	viewerRole := makeRole("viewer", []roledom.Permission{
		{Provider: "seca.compute", Resources: []string{"instances"}, Verb: []string{"list"}},
	})
	validAssignment := makeAssignment([]string{"viewer"}, []radom.RoleAssignmentScope{allScope})

	baseClaim := authzport.AuthorizationClaim{
		Subject:  "alice",
		Provider: "seca.compute",
		Resource: "instances",
		Verb:     "list",
		Tenant:   "t1",
		Roles:    []string{"viewer"},
	}

	apiServerUnavailable := errors.New("api server unavailable")

	tests := []struct {
		name         string
		roles        []*roledom.Role
		assignments  []*radom.RoleAssignment
		roleErr      error
		assignErr    error
		claim        authzport.AuthorizationClaim
		wantDecision authzport.Decision
		wantKind     *kernel.ErrKind // nil means no domain error expected
	}{
		{
			name:         "allow: matching role and assignment → DecisionAllowed",
			roles:        []*roledom.Role{viewerRole},
			assignments:  []*radom.RoleAssignment{validAssignment},
			claim:        baseClaim,
			wantDecision: authzport.DecisionAllowed,
		},
		{
			name:         "deny: no matching assignment → DecisionDenied with ErrForbidden",
			roles:        []*roledom.Role{viewerRole},
			assignments:  nil,
			claim:        baseClaim,
			wantDecision: authzport.DecisionDenied,
			wantKind:     new(kernel.KindForbidden),
		},
		{
			// A RoleAssignment whose Subs list names only "alice" must deny "bob" even
			// when the scope, roles, and permissions all match — proving that subject
			// filtering is enforced by the real Evaluate function through the Checker.
			name:         "deny: subject not in Subs → DecisionDenied with ErrForbidden",
			roles:        []*roledom.Role{viewerRole},
			assignments:  []*radom.RoleAssignment{assignSubs([]string{"alice"}, []string{"viewer"}, allScope)},
			claim:        with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = "bob" }),
			wantDecision: authzport.DecisionDenied,
			wantKind:     new(kernel.KindForbidden),
		},
		{
			// Key safety property: an unreachable role store must NOT be reported
			// as DecisionDenied (which would silently block legitimate users while
			// hiding the infrastructure outage). It must be DecisionError → HTTP 500.
			name:         "technical: role reader error → DecisionError, not DecisionDenied",
			roleErr:      apiServerUnavailable,
			claim:        baseClaim,
			wantDecision: authzport.DecisionError,
			wantKind:     new(kernel.KindInternal),
		},
		{
			name:         "technical: assignment reader error → DecisionError, not DecisionDenied",
			roles:        []*roledom.Role{viewerRole},
			assignErr:    apiServerUnavailable,
			claim:        baseClaim,
			wantDecision: authzport.DecisionError,
			wantKind:     new(kernel.KindInternal),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			checker := NewChecker(
				&stubRoleReader{roles: tc.roles, err: tc.roleErr},
				&stubAssignmentReader{assignments: tc.assignments, err: tc.assignErr},
				discardLog(),
			)

			decision, err := checker.Authorize(context.Background(), tc.claim)

			if decision != tc.wantDecision {
				t.Errorf("decision = %v, want %v", decision, tc.wantDecision)
			}

			if tc.wantKind == nil {
				if err != nil {
					t.Errorf("err = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			ke := kernel.AsError(err)
			if ke == nil || ke.Kind != *tc.wantKind {
				t.Errorf("err kind = %v, want %v (err: %v)", ke, *tc.wantKind, err)
			}
		})
	}
}
