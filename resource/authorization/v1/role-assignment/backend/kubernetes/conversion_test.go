package kubernetes_test

import (
	"slices"
	"strings"
	"testing"

	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

	. "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment/backend/kubernetes"
)

// FuzzRoleAssignmentSpecRoundTrip verifies that a RoleAssignment domain survives a
// domain→CR→domain→CR→domain round-trip. The spec carries only string slices
// (subs, roles) and scopes (tenants, regions, workspaces), so the conversion is a
// straight copy; the only normalisation happens on Provider (slash↔underscore in the
// internal label). Hence we compare the *second* and *third* passes for stability.
//
// Invariants (domain2 == domain3):
//   - Name, Provider, Tenant, Region are stable after one round-trip. Role assignments
//     are tenant-scoped, so Workspace is never set.
//   - Subs, Roles and the per-scope slices are preserved exactly.
func FuzzRoleAssignmentSpecRoundTrip(f *testing.F) {
	// (sub, role, scopeTenant, scopeRegion, scopeWorkspace, name, provider, tenant, region)
	f.Add("user1@example.com", "workspace-viewer", "t-1", "de-fra", "ws-1", "ra-1", "seca.authorization/v1", "t-1", "de-fra")
	f.Add("", "", "", "", "", "", "", "", "")
	f.Add("service-account-1", "project-manager", "*", "", "", "ra", "a/_b", "t", "")
	f.Add(strings.Repeat("s", 128), strings.Repeat("r", 64), "t", "r", "w", strings.Repeat("n", 64), "provider/x", "t", "de")

	f.Fuzz(func(t *testing.T,
		sub, role, scopeTenant, scopeRegion, scopeWorkspace string,
		name, provider, tenant, region string,
	) {
		domain := &radom.RoleAssignment{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name:     name,
					Provider: provider,
				},
				Scope:  kernelresource.Scope{Tenant: tenant},
				Region: region,
			},
			Spec: radom.RoleAssignmentSpec{
				Subs: []string{sub},
				Scopes: []radom.RoleAssignmentScope{
					{
						Tenants:    []string{scopeTenant},
						Regions:    []string{scopeRegion},
						Workspaces: []string{scopeWorkspace},
					},
				},
				Roles: []string{role},
			},
		}

		cr1, err := RoleAssignmentToCR(domain)
		if err != nil {
			return
		}

		domain2, err := RoleAssignmentFromCR(cr1)
		if err != nil {
			t.Errorf("CR→domain failed after successful domain→CR: %v", err)
			return
		}

		cr2, err := RoleAssignmentToCR(domain2)
		if err != nil {
			t.Errorf("second domain→CR failed: %v", err)
			return
		}

		domain3, err := RoleAssignmentFromCR(cr2)
		if err != nil {
			t.Errorf("second CR→domain failed: %v", err)
			return
		}

		// CommonMetadata: stable after one round-trip.
		if domain2.Name != domain3.Name {
			t.Errorf("Name not stable: %q → %q", domain2.Name, domain3.Name)
		}
		if domain2.Provider != domain3.Provider {
			t.Errorf("Provider not stable: %q → %q", domain2.Provider, domain3.Provider)
		}
		if domain2.Tenant != domain3.Tenant {
			t.Errorf("Tenant not stable: %q → %q", domain2.Tenant, domain3.Tenant)
		}
		if domain2.Region != domain3.Region {
			t.Errorf("Region not stable: %q → %q", domain2.Region, domain3.Region)
		}
		if domain2.Workspace != "" {
			t.Errorf("Workspace must never be set for a tenant-scoped role assignment, got %q", domain2.Workspace)
		}

		// Spec slices: preserved exactly across the round-trip.
		if !slices.Equal(domain2.Spec.Subs, domain3.Spec.Subs) {
			t.Errorf("Subs not stable: %v → %v", domain2.Spec.Subs, domain3.Spec.Subs)
		}
		if !slices.Equal(domain2.Spec.Roles, domain3.Spec.Roles) {
			t.Errorf("Roles not stable: %v → %v", domain2.Spec.Roles, domain3.Spec.Roles)
		}
		if len(domain2.Spec.Scopes) != len(domain3.Spec.Scopes) {
			t.Fatalf("Scopes length not stable: %d → %d", len(domain2.Spec.Scopes), len(domain3.Spec.Scopes))
		}
		for i := range domain2.Spec.Scopes {
			s2, s3 := domain2.Spec.Scopes[i], domain3.Spec.Scopes[i]
			if !slices.Equal(s2.Tenants, s3.Tenants) {
				t.Errorf("Scope[%d].Tenants not stable: %v → %v", i, s2.Tenants, s3.Tenants)
			}
			if !slices.Equal(s2.Regions, s3.Regions) {
				t.Errorf("Scope[%d].Regions not stable: %v → %v", i, s2.Regions, s3.Regions)
			}
			if !slices.Equal(s2.Workspaces, s3.Workspaces) {
				t.Errorf("Scope[%d].Workspaces not stable: %v → %v", i, s2.Workspaces, s3.Workspaces)
			}
		}
	})
}
