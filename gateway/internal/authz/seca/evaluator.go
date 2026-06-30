// Package seca implements the SECA RBAC authorization checker for the ECP gateway.
//
// The package provides two implementations of authzport.Checker:
//   - Checker: fetches Roles and RoleAssignments from Kubernetes on every request.
//   - CachedChecker: same logic but reads from an in-process Kubernetes informer cache
//     (see cache.go) to avoid API-server round-trips on the hot path.
//
// Both implementations delegate the actual policy evaluation to the pure Evaluate
// function defined in this file.
package seca

import (
	"slices"
	"strings"

	"github.com/gobwas/glob"

	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
)

// Evaluate checks whether the AuthorizationClaim is permitted by the supplied
// roles and assignments.
//
// Authorization algorithm (locked in design review):
//
//	authorized = ∃ ra ∈ assignments:
//	    scopeCovers(ra.Scopes, tenant, region, workspace)
//	  ∧ ∃ roleName ∈ (ra.Roles ∩ claim.Roles):
//	        role := rolesByName[roleName]
//	        ∃ p ∈ role.Permissions:
//	            p.Provider == claim.Provider
//	          ∧ matchResource(p.Resources, claim.Resource, claim.Name)
//	          ∧ matchVerb(p.Verb, claim.Verb)
//
// claim.Roles are SECA Role names (not subject identifiers).
// RoleAssignment.Scopes scope the grant; empty Tenants/Regions/Workspaces = wildcard.
func Evaluate(
	claim authzport.AuthorizationClaim,
	rolesByName map[string]*roledom.Role,
	assignments []*radom.RoleAssignment,
) bool {
	for _, ra := range assignments {
		if !assignmentCoversScope(ra, claim.Tenant, claim.Region, claim.Workspace) {
			continue
		}
		for _, roleName := range ra.Spec.Roles {
			if !claimHasRole(claim.Roles, roleName) {
				continue
			}
			role, ok := rolesByName[roleName]
			if !ok {
				continue
			}
			for _, p := range role.Spec.Permissions {
				if p.Provider == claim.Provider &&
					matchResource(p.Resources, claim.Resource, claim.Name) &&
					matchVerb(p.Verb, claim.Verb) {
					return true
				}
			}
		}
	}
	return false
}

// claimHasRole reports whether the claim carries the named SECA Role.
func claimHasRole(claimRoles []string, roleName string) bool {
	return slices.Contains(claimRoles, roleName)
}

// assignmentCoversScope reports whether any of the RoleAssignment's scopes
// covers the requested (tenant, region, workspace) combination.
//
// Scope matching rules (from the RoleAssignment CRD spec):
//   - Empty Tenants = match any tenant (within the assignment's namespace).
//   - Empty Regions = match any region.
//   - Empty Workspaces = match any workspace (or no workspace).
//
// A single scope entry grants access when ALL three dimensions match.
// The overall assignment grants access when AT LEAST ONE scope entry matches.
func assignmentCoversScope(ra *radom.RoleAssignment, tenant, region, workspace string) bool {
	for _, scope := range ra.Spec.Scopes {
		if sliceCovers(scope.Tenants, tenant) &&
			sliceCovers(scope.Regions, region) &&
			sliceCovers(scope.Workspaces, workspace) {
			return true
		}
	}
	return false
}

// sliceCovers reports whether the list "covers" the given value.
// An empty list is a wildcard (covers everything); a non-empty list must contain value.
func sliceCovers(list []string, value string) bool {
	if len(list) == 0 {
		return true
	}
	return slices.Contains(list, value)
}

// matchResource reports whether any pattern in the Permission.Resources list
// authorizes the (resource, name) pair carried by the claim.
//
// Matching target:
//   - Item operations (name != ""): "resource/name" — e.g. "instances/inst1".
//   - Collection operations (name == ""): "resource" — e.g. "instances".
//
// Each pattern is interpreted as a gobwas/glob expression. The default glob
// configuration (no separator restriction) allows "*" to match across slashes,
// making it a universal wildcard ("*" covers both "instances" and "instances/inst1").
// Provider-specific patterns such as "instances/*" match only item operations.
func matchResource(patterns []string, resource, name string) bool {
	var target string
	if name != "" {
		target = resource + "/" + name
	} else {
		target = resource
	}

	for _, pattern := range patterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			// Invalid glob patterns are treated as non-matching; this should not
			// occur with well-formed CRDs but avoids a panic on bad data.
			continue
		}
		if g.Match(target) {
			return true
		}
	}
	return false
}

// matchVerb reports whether any pattern in the Permission.Verb list authorizes
// the requested verb.
//
// Verb matching rules (from the RoleAssignment CRD spec):
//   - "*" matches any verb.
//   - Exact match: "get" matches "get".
//   - Bare verb covers actions: "post" matches "post.start", "post.restart", etc.
//     The convention is that a bare verb without a dot is a prefix for all
//     sub-actions of that verb (e.g. Permission.Verb=["post"] grants all POST actions).
func matchVerb(patterns []string, verb string) bool {
	for _, p := range patterns {
		if p == "*" || p == verb {
			return true
		}
		// Bare verb covers "verb.action" (e.g. "post" covers "post.start").
		if !strings.ContainsRune(p, '.') && strings.HasPrefix(verb, p+".") {
			return true
		}
	}
	return false
}
