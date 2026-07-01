package seca

import (
	"testing"

	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	commondom "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

const instanceName = "inst1"

// makeRole is a test helper that builds a *roledom.Role.
func makeRole(name string, permissions []roledom.Permission) *roledom.Role {
	return &roledom.Role{
		GlobalTenantMetadata: commondom.GlobalTenantMetadata{
			CommonMetadata: commondom.CommonMetadata{Name: name},
		},
		Spec: roledom.RoleSpec{Permissions: permissions},
	}
}

// makeAssignment is a test helper that builds a *radom.RoleAssignment.
// Subs defaults to ["*"] so that existing test cases (which focus on scope/role/permission
// correctness and are intentionally subject-agnostic) remain unaffected.
func makeAssignment(roles []string, scopes []radom.RoleAssignmentScope) *radom.RoleAssignment {
	return &radom.RoleAssignment{
		Spec: radom.RoleAssignmentSpec{
			Subs:   []string{"*"},
			Roles:  roles,
			Scopes: scopes,
		},
	}
}

// assignSubs is like makeAssignment but lets the caller control the Subs list,
// for tests that exercise subject-based filtering.
func assignSubs(subs, roles []string, scopes ...radom.RoleAssignmentScope) *radom.RoleAssignment {
	return &radom.RoleAssignment{
		Spec: radom.RoleAssignmentSpec{
			Subs:   subs,
			Roles:  roles,
			Scopes: scopes,
		},
	}
}

// allScope is a scope that covers all tenants/regions/workspaces (all slices empty).
var allScope = radom.RoleAssignmentScope{}

// tenantScope is a scope covering only a specific tenant.
func tenantScope(tenant string) radom.RoleAssignmentScope {
	return radom.RoleAssignmentScope{Tenants: []string{tenant}}
}

// tenantRegionScope is a scope covering a specific tenant and region.
func tenantRegionScope(tenant, region string) radom.RoleAssignmentScope {
	return radom.RoleAssignmentScope{Tenants: []string{tenant}, Regions: []string{region}}
}

func TestEvaluate(t *testing.T) {
	t.Parallel()

	viewerRole := makeRole("viewer", []roledom.Permission{
		{Provider: "seca.compute", Resources: []string{"instances"}, Verb: []string{"get", "list"}},
	})
	// adminRole uses Resources: ["*"] (wildcard) so it covers both collection and item operations.
	adminRole := makeRole("admin", []roledom.Permission{
		{Provider: "seca.compute", Resources: []string{"*"}, Verb: []string{"*"}},
		{Provider: "seca.network", Resources: []string{"*"}, Verb: []string{"*"}},
	})
	wildcardRole := makeRole("all-access", []roledom.Permission{
		{Provider: "seca.compute", Resources: []string{"*"}, Verb: []string{"*"}},
	})

	rolesByName := map[string]*roledom.Role{
		"viewer":     viewerRole,
		"admin":      adminRole,
		"all-access": wildcardRole,
	}

	baseClaim := authzport.AuthorizationClaim{
		Provider:  "seca.compute",
		Resource:  "instances",
		Name:      "",
		Verb:      "list",
		Tenant:    "t1",
		Region:    "r1",
		Workspace: "w1",
	}

	assign := func(roles []string, scopes ...radom.RoleAssignmentScope) *radom.RoleAssignment {
		return makeAssignment(roles, scopes)
	}

	tests := []struct {
		name        string
		claim       authzport.AuthorizationClaim
		assignments []*radom.RoleAssignment
		want        bool
	}{
		// ── Basic allow/deny ──────────────────────────────────────────────────
		{
			name:        "exact match: viewer can list instances",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{"viewer"}, allScope)},
			want:        true,
		},
		{
			name:        "exact match: viewer can get instance",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Name = instanceName; c.Verb = "get" }),
			assignments: []*radom.RoleAssignment{assign([]string{"viewer"}, allScope)},
			want:        false, // "instances" pattern != "instances/inst1"
		},
		{
			name:        "wildcard resource: admin can get instance",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Name = instanceName; c.Verb = "get" }),
			assignments: []*radom.RoleAssignment{assign([]string{"admin"}, allScope)},
			want:        true, // admin has Verb "*" on instances
		},
		{
			name:        "wildcard resource role: all-access can get instance",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Name = instanceName; c.Verb = "get" }),
			assignments: []*radom.RoleAssignment{assign([]string{"all-access"}, allScope)},
			want:        true, // Resources ["*"] with glob matches "instances/inst1"
		},
		{
			name:        "no matching role name in claim",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Roles = []string{"other-role"} }),
			assignments: []*radom.RoleAssignment{assign([]string{"viewer"}, allScope)},
			want:        false,
		},
		{
			name:        "claim has role but role not in rolesByName",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Roles = []string{"nonexistent"} }),
			assignments: []*radom.RoleAssignment{assign([]string{"nonexistent"}, allScope)},
			want:        false,
		},
		{
			name:        "provider mismatch",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Provider = "seca.storage" }),
			assignments: []*radom.RoleAssignment{assign([]string{"viewer"}, allScope)},
			want:        false,
		},
		{
			name:        "wrong verb denied",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Verb = "delete" }),
			assignments: []*radom.RoleAssignment{assign([]string{"viewer"}, allScope)},
			want:        false,
		},
		{
			name:        "empty assignments → denied",
			claim:       baseClaim,
			assignments: nil,
			want:        false,
		},
		// ── Scope matching ────────────────────────────────────────────────────
		{
			name:        "scope covers tenant",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{"viewer"}, tenantScope("t1"))},
			want:        true,
		},
		{
			name:        "scope wrong tenant",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{"viewer"}, tenantScope("t2"))},
			want:        false,
		},
		{
			name:        "scope empty region = wildcard",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{"viewer"}, tenantScope("t1"))},
			want:        true,
		},
		{
			name:        "scope specific region matches",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{"viewer"}, tenantRegionScope("t1", "r1"))},
			want:        true,
		},
		{
			name:        "scope specific region mismatch",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{"viewer"}, tenantRegionScope("t1", "r2"))},
			want:        false,
		},
		{
			name:  "scope empty workspace = wildcard",
			claim: baseClaim, // workspace="w1"
			assignments: []*radom.RoleAssignment{
				assign([]string{"viewer"}, radom.RoleAssignmentScope{Tenants: []string{"t1"}, Workspaces: []string{}}),
			},
			want: true,
		},
		{
			name:  "scope workspace mismatch",
			claim: baseClaim, // workspace="w1"
			assignments: []*radom.RoleAssignment{
				assign([]string{"viewer"}, radom.RoleAssignmentScope{Tenants: []string{"t1"}, Workspaces: []string{"w2"}}),
			},
			want: false,
		},
		// ── Verb and resource matching (via wildcard admin role) ──────────────
		{
			name:        "verb '*' allows any verb: admin can delete instance",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Name = instanceName; c.Verb = "delete" }),
			assignments: []*radom.RoleAssignment{assign([]string{"admin"}, allScope)},
			want:        true,
		},
		{
			name:        "wildcard resource: admin can get named instance",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Name = instanceName; c.Verb = "get" }),
			assignments: []*radom.RoleAssignment{assign([]string{"admin"}, allScope)},
			want:        true, // admin Resources=["*"] covers "instances/inst1"
		},
		// ── Multiple assignments (OR semantics) ───────────────────────────────
		{
			name:  "second assignment covers when first does not",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Roles = []string{"viewer"} }),
			assignments: []*radom.RoleAssignment{
				assign([]string{"viewer"}, tenantScope("t2")), // wrong tenant
				assign([]string{"viewer"}, tenantScope("t1")), // correct tenant
			},
			want: true,
		},
		// ── Subject matching ──────────────────────────────────────────────────
		{
			name:  "subject exact match → allowed",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = "alice" }),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{"alice"}, []string{"viewer"}, allScope),
			},
			want: true,
		},
		{
			name:  "subject mismatch → denied",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = "bob" }),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{"alice"}, []string{"viewer"}, allScope),
			},
			want: false,
		},
		{
			name:  "wildcard subject '*' → allowed for any caller",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = "anyone" }),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{"*"}, []string{"viewer"}, allScope),
			},
			want: true,
		},
		{
			name:  "empty subs → denied (fail-closed, not a wildcard)",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = "alice" }),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{}, []string{"viewer"}, allScope),
			},
			want: false,
		},
		{
			name:  "multi-subject list: second entry matches → allowed",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = "carol" }),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{"alice", "carol"}, []string{"viewer"}, allScope),
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Inject roles into claim if not already set by with().
			claim := tc.claim
			if len(claim.Roles) == 0 {
				// Default to roles that appear in the first assignment (for convenience).
				if len(tc.assignments) > 0 {
					claim.Roles = tc.assignments[0].Spec.Roles
				}
			}

			got := Evaluate(claim, rolesByName, tc.assignments)
			if got != tc.want {
				t.Errorf("Evaluate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// with is a small builder helper: copies the claim, applies the mutator, returns the copy.
func with(c authzport.AuthorizationClaim, mutate func(*authzport.AuthorizationClaim)) authzport.AuthorizationClaim {
	mutate(&c)
	return c
}

func TestMatchVerb(t *testing.T) {
	t.Parallel()
	tests := []struct {
		patterns []string
		verb     string
		want     bool
	}{
		{[]string{"get"}, "get", true},
		{[]string{"get"}, "list", false},
		{[]string{"*"}, "delete", true},
		{[]string{"*"}, "post.restart", true},
		{[]string{"post"}, "post.restart", true},
		{[]string{"post"}, "post.start", true},
		{[]string{"post"}, "post", true},
		{[]string{"post.start"}, "post.restart", false},
		{[]string{"post.start"}, "post.start", true},
		{[]string{"get", "list"}, "list", true},
		{[]string{"get", "list"}, "put", false},
	}
	for _, tc := range tests {
		got := matchVerb(tc.patterns, tc.verb)
		if got != tc.want {
			t.Errorf("matchVerb(%v, %q) = %v, want %v", tc.patterns, tc.verb, got, tc.want)
		}
	}
}

func TestMatchResource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		patterns []string
		resource string
		name     string
		want     bool
	}{
		{[]string{"instances"}, "instances", "", true},
		{[]string{"instances"}, "instances", instanceName, false}, // exact "instances" != "instances/inst1"
		{[]string{"instances/*"}, "instances", instanceName, true},
		{[]string{"instances/*"}, "instances", "", false}, // "instances/*" requires a name
		{[]string{"*"}, "instances", instanceName, true},  // "*" matches across "/"
		{[]string{"*"}, "networks/subnets", "sub1", true},
		{[]string{"networks/subnets"}, "networks/subnets", "", true},
		{[]string{"networks/subnets"}, "networks", "", false},
	}
	for _, tc := range tests {
		got := matchResource(tc.patterns, tc.resource, tc.name)
		if got != tc.want {
			t.Errorf("matchResource(%v, %q, %q) = %v, want %v", tc.patterns, tc.resource, tc.name, got, tc.want)
		}
	}
}

func TestSliceCovers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		list  []string
		value string
		want  bool
	}{
		{nil, "anything", true},        // empty = wildcard
		{[]string{}, "anything", true}, // empty slice = wildcard
		{[]string{"t1"}, "t1", true},
		{[]string{"t1"}, "t2", false},
		{[]string{"t1", "t2"}, "t2", true},
	}
	for _, tc := range tests {
		got := sliceCovers(tc.list, tc.value)
		if got != tc.want {
			t.Errorf("sliceCovers(%v, %q) = %v, want %v", tc.list, tc.value, got, tc.want)
		}
	}
}

func TestSubsGrant(t *testing.T) {
	t.Parallel()
	tests := []struct {
		subs    []string
		subject string
		want    bool
	}{
		{[]string{"*"}, "alice", true},                  // wildcard covers any subject
		{[]string{"*"}, "", true},                       // wildcard covers empty subject too
		{[]string{"alice"}, "alice", true},              // exact match
		{[]string{"alice"}, "bob", false},               // mismatch
		{[]string{"alice", "bob"}, "bob", true},         // second entry matches
		{[]string{"alice", "bob"}, "carol", false},      // no entry matches
		{nil, "alice", false},                           // nil subs → deny (fail-closed)
		{[]string{}, "alice", false},                    // empty subs → deny (not a wildcard)
		{[]string{"alice", "*"}, "anyone", true},        // wildcard in a mixed list
	}
	for _, tc := range tests {
		got := subsGrant(tc.subs, tc.subject)
		if got != tc.want {
			t.Errorf("subsGrant(%v, %q) = %v, want %v", tc.subs, tc.subject, got, tc.want)
		}
	}
}
