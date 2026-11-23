package seca

import (
	"testing"

	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	commondom "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

const instanceName = "inst1"

// Shared test fixtures for provider/resource/verb/role/subject literals used across this
// package's tests.
const (
	providerCompute   = "seca.compute"
	resourceInstances = "instances"
	verbList          = "list"
	verbGet           = "get"
	roleViewer        = "viewer"
	roleAdmin         = "admin"
	subjectAlice      = "alice"
	subjectBob        = "bob"
)

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

	viewerRole := makeRole(roleViewer, []roledom.Permission{
		{Provider: providerCompute, Resources: []string{resourceInstances}, Verb: []string{verbGet, verbList}},
	})
	// adminRole uses Resources: ["*"] (wildcard) so it covers both collection and item operations.
	adminRole := makeRole(roleAdmin, []roledom.Permission{
		{Provider: providerCompute, Resources: []string{"*"}, Verb: []string{"*"}},
		{Provider: "seca.network", Resources: []string{"*"}, Verb: []string{"*"}},
	})
	wildcardRole := makeRole("all-access", []roledom.Permission{
		{Provider: providerCompute, Resources: []string{"*"}, Verb: []string{"*"}},
	})

	rolesByName := map[string]*roledom.Role{
		roleViewer:   viewerRole,
		roleAdmin:    adminRole,
		"all-access": wildcardRole,
	}

	baseClaim := authzport.AuthorizationClaim{
		Provider:  providerCompute,
		Resource:  resourceInstances,
		Name:      "",
		Verb:      verbList,
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
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        true,
		},
		{
			name:        "exact match: viewer can get instance",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Name = instanceName; c.Verb = verbGet }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        false, // "instances" pattern != "instances/inst1"
		},
		{
			name:        "wildcard resource: admin can get instance",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Name = instanceName; c.Verb = verbGet }),
			assignments: []*radom.RoleAssignment{assign([]string{roleAdmin}, allScope)},
			want:        true, // admin has Verb "*" on instances
		},
		{
			name:        "wildcard resource role: all-access can get instance",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Name = instanceName; c.Verb = verbGet }),
			assignments: []*radom.RoleAssignment{assign([]string{"all-access"}, allScope)},
			want:        true, // Resources ["*"] with glob matches "instances/inst1"
		},
		{
			name:        "assignment role not defined in rolesByName → denied",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{"nonexistent"}, allScope)},
			want:        false,
		},
		{
			name:        "provider mismatch",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Provider = "seca.storage" }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        false,
		},
		{
			name:        "wrong verb denied",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Verb = "delete" }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
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
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, tenantScope("t1"))},
			want:        true,
		},
		{
			name:        "scope wrong tenant",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, tenantScope("t2"))},
			want:        false,
		},
		{
			name:        "scope empty region = wildcard",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, tenantScope("t1"))},
			want:        true,
		},
		{
			name:        "scope specific region matches",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, tenantRegionScope("t1", "r1"))},
			want:        true,
		},
		{
			name:        "scope specific region mismatch",
			claim:       baseClaim,
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, tenantRegionScope("t1", "r2"))},
			want:        false,
		},
		{
			name:  "scope empty workspace = wildcard",
			claim: baseClaim, // workspace="w1"
			assignments: []*radom.RoleAssignment{
				assign([]string{roleViewer}, radom.RoleAssignmentScope{Tenants: []string{"t1"}, Workspaces: []string{}}),
			},
			want: true,
		},
		{
			name:  "scope workspace mismatch",
			claim: baseClaim, // workspace="w1"
			assignments: []*radom.RoleAssignment{
				assign([]string{roleViewer}, radom.RoleAssignmentScope{Tenants: []string{"t1"}, Workspaces: []string{"w2"}}),
			},
			want: false,
		},
		// ── Verb and resource matching (via wildcard admin role) ──────────────
		{
			name:        "verb '*' allows any verb: admin can delete instance",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Name = instanceName; c.Verb = "delete" }),
			assignments: []*radom.RoleAssignment{assign([]string{roleAdmin}, allScope)},
			want:        true,
		},
		{
			name:        "wildcard resource: admin can get named instance",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Name = instanceName; c.Verb = verbGet }),
			assignments: []*radom.RoleAssignment{assign([]string{roleAdmin}, allScope)},
			want:        true, // admin Resources=["*"] covers "instances/inst1"
		},
		// ── Multiple assignments (OR semantics) ───────────────────────────────
		{
			name:  "second assignment covers when first does not",
			claim: baseClaim,
			assignments: []*radom.RoleAssignment{
				assign([]string{roleViewer}, tenantScope("t2")), // wrong tenant
				assign([]string{roleViewer}, tenantScope("t1")), // correct tenant
			},
			want: true,
		},
		// ── Subject matching ──────────────────────────────────────────────────
		{
			name:  "subject exact match → allowed",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = subjectAlice }),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{subjectAlice}, []string{roleViewer}, allScope),
			},
			want: true,
		},
		{
			name:  "subject mismatch → denied",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = subjectBob }),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{subjectAlice}, []string{roleViewer}, allScope),
			},
			want: false,
		},
		{
			name:  "wildcard subject '*' → allowed for any caller",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = "anyone" }),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{"*"}, []string{roleViewer}, allScope),
			},
			want: true,
		},
		{
			name:  "empty subs → denied (fail-closed, not a wildcard)",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = subjectAlice }),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{}, []string{roleViewer}, allScope),
			},
			want: false,
		},
		{
			name:  "multi-subject list: second entry matches → allowed",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) { c.Subject = "carol" }),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{subjectAlice, "carol"}, []string{roleViewer}, allScope),
			},
			want: true,
		},
		// ── Token down-scoping (caps that only narrow) ────────────────────────
		{
			name:        "down-scope tenant covers request",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.TokenScope.Tenants = []string{"t1"} }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        true,
		},
		{
			name:        "down-scope tenant excludes request → denied",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.TokenScope.Tenants = []string{"t2"} }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        false,
		},
		{
			name:        "down-scope region covers request",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.TokenScope.Regions = []string{"r1"} }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        true,
		},
		{
			name:        "down-scope region excludes request → denied",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.TokenScope.Regions = []string{"r2"} }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        false,
		},
		{
			name:        "down-scope workspace covers request",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.TokenScope.Workspaces = []string{"w1"} }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        true,
		},
		{
			name:        "down-scope workspace excludes request → denied",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.TokenScope.Workspaces = []string{"w2"} }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        false,
		},
		{
			name: "down-scope region skipped when request region empty",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) {
				c.Region = ""
				c.TokenScope.Regions = []string{"r1"}
			}),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        true, // empty request region ⇒ cap not applicable
		},
		{
			name: "down-scope denies even when RBAC would allow",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) {
				c.Subject = subjectAlice
				c.TokenScope.Tenants = []string{"other"}
			}),
			assignments: []*radom.RoleAssignment{
				assignSubs([]string{subjectAlice}, []string{roleAdmin}, allScope),
			},
			want: false, // admin grants everything, but the token cap excludes t1
		},

		// ── Issuer-asserted tenant membership (claim.MemberTenants) ───────────
		{
			name:        "membership covers request tenant",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.MemberTenants = []string{"t1", "t9"} }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        true,
		},
		{
			name:        "membership excludes request tenant → denied",
			claim:       with(baseClaim, func(c *authzport.AuthorizationClaim) { c.MemberTenants = []string{"t9"} }),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        false,
		},
		{
			name: "membership gates a subs:[*] wildcard grant the token scope cannot",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) {
				c.Subject = "outsider"
				c.MemberTenants = []string{"t9"}
			}),
			// assign() builds Subs:["*"], so without the membership gate every
			// authenticated caller would be granted admin here.
			assignments: []*radom.RoleAssignment{assign([]string{roleAdmin}, allScope)},
			want:        false,
		},
		{
			name: "membership and token scope are both enforced (intersection)",
			claim: with(baseClaim, func(c *authzport.AuthorizationClaim) {
				c.MemberTenants = []string{"t1", "t2"}
				c.TokenScope.Tenants = []string{"t2"}
			}),
			assignments: []*radom.RoleAssignment{assign([]string{roleViewer}, allScope)},
			want:        false, // member of t1, but the token narrowed itself to t2
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := Evaluate(tc.claim, rolesByName, tc.assignments)
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
		{[]string{verbGet}, verbGet, true},
		{[]string{verbGet}, verbList, false},
		{[]string{"*"}, "delete", true},
		{[]string{"*"}, "post.restart", true},
		{[]string{"post"}, "post.restart", true},
		{[]string{"post"}, "post.start", true},
		{[]string{"post"}, "post", true},
		{[]string{"post.start"}, "post.restart", false},
		{[]string{"post.start"}, "post.start", true},
		{[]string{verbGet, verbList}, verbList, true},
		{[]string{verbGet, verbList}, "put", false},
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
		{[]string{resourceInstances}, resourceInstances, "", true},
		{[]string{resourceInstances}, resourceInstances, instanceName, false}, // exact resourceInstances != "instances/inst1"
		{[]string{"instances/*"}, resourceInstances, instanceName, true},
		{[]string{"instances/*"}, resourceInstances, "", false}, // "instances/*" requires a name
		{[]string{"*"}, resourceInstances, instanceName, true},  // "*" matches across "/"
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

func TestTokenScopeCovers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		list  []string
		value string
		want  bool
	}{
		{nil, "t1", true},             // no cap ⇒ unconstrained
		{[]string{}, "t1", true},      // empty cap ⇒ unconstrained
		{[]string{"t1"}, "t1", true},  // listed ⇒ allowed
		{[]string{"t1"}, "t2", false}, // not listed ⇒ denied
		{[]string{"t1"}, "", true},    // empty request value ⇒ dimension not applicable
		{[]string{"t1", "t2"}, "t2", true},
	}
	for _, tc := range tests {
		got := tokenScopeCovers(tc.list, tc.value)
		if got != tc.want {
			t.Errorf("tokenScopeCovers(%v, %q) = %v, want %v", tc.list, tc.value, got, tc.want)
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
		{[]string{"*"}, subjectAlice, true},                    // wildcard covers any subject
		{[]string{"*"}, "", true},                              // wildcard covers empty subject too
		{[]string{subjectAlice}, subjectAlice, true},           // exact match
		{[]string{subjectAlice}, subjectBob, false},            // mismatch
		{[]string{subjectAlice, subjectBob}, subjectBob, true}, // second entry matches
		{[]string{subjectAlice, subjectBob}, "carol", false},   // no entry matches
		{nil, subjectAlice, false},                             // nil subs → deny (fail-closed)
		{[]string{}, subjectAlice, false},                      // empty subs → deny (not a wildcard)
		{[]string{subjectAlice, "*"}, "anyone", true},          // wildcard in a mixed list
	}
	for _, tc := range tests {
		got := subsGrant(tc.subs, tc.subject)
		if got != tc.want {
			t.Errorf("subsGrant(%v, %q) = %v, want %v", tc.subs, tc.subject, got, tc.want)
		}
	}
}
