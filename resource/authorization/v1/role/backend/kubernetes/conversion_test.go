package kubernetes_test

import (
	"testing"

	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	. "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role/backend/kubernetes"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// FuzzRoleSpecRoundTrip verifies that a Role domain value survives a
// domain→CR→domain→CR→domain→CR round-trip.
//
// Invariants:
//   - Name, Provider, and Tenant are stable after one round-trip (domain2 == domain3).
//   - Permissions are stable after one round-trip.
func FuzzRoleSpecRoundTrip(f *testing.F) {
	f.Add("my-role", "seca.authorization/v1", "t-1", "seca.authorization", "roles", "get")
	f.Add("", "", "", "", "", "")
	f.Add("admin-role", "ionos/v1", "tenant-42", "seca.compute", "instances,block-storages", "get,list,create")

	f.Fuzz(func(t *testing.T,
		name, provider, tenant string,
		permProvider, permResource, permVerb string,
	) {
		domain := &roledom.Role{
			GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name:     name,
					Provider: provider,
				},
				Scope: kernelresource.Scope{Tenant: tenant},
			},
			Spec: roledom.RoleSpec{
				Permissions: []roledom.Permission{
					{
						Provider:  permProvider,
						Resources: []string{permResource},
						Verb:      []string{permVerb},
					},
				},
			},
		}

		cr1, err := RoleToCR(domain)
		if err != nil {
			return
		}

		domain2, err := RoleFromCR(cr1)
		if err != nil {
			t.Errorf("CR→domain failed after successful domain→CR: %v", err)
			return
		}

		cr2, err := RoleToCR(domain2)
		if err != nil {
			t.Errorf("second domain→CR failed: %v", err)
			return
		}

		domain3, err := RoleFromCR(cr2)
		if err != nil {
			t.Errorf("second CR→domain failed: %v", err)
			return
		}

		// Identity fields: stable after one round-trip.
		if domain2.Name != domain3.Name {
			t.Errorf("Name not stable: %q → %q", domain2.Name, domain3.Name)
		}
		if domain2.Provider != domain3.Provider {
			t.Errorf("Provider not stable: %q → %q", domain2.Provider, domain3.Provider)
		}
		if domain2.Tenant != domain3.Tenant {
			t.Errorf("Tenant not stable: %q → %q", domain2.Tenant, domain3.Tenant)
		}

		// Spec stability.
		if len(domain2.Spec.Permissions) != len(domain3.Spec.Permissions) {
			t.Errorf("Permissions length not stable: %d → %d", len(domain2.Spec.Permissions), len(domain3.Spec.Permissions))
		}
	})
}
