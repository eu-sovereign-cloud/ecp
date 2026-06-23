package kubernetes_test

import (
	"strings"
	"testing"

	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"

	. "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/backend/kubernetes"
)

// FuzzBlockStorageSpecRoundTrip verifies that a BlockStorage domain survives a
// domain→CR→domain→CR→domain→CR round-trip. The interesting complexity is in
// mapDomainReferenceToCR, which extracts "providers/", "regions/",
// "tenants/", "workspaces/" from the Resource path — a normalizing transformation.
//
// Invariants:
//   - SizeGB, Name, Provider, Tenant, Workspace, Region are stable after one round-trip
//     (domain2 == domain3).
//   - SkuRef CR fields are compared at cr2 vs cr3: the first pass normalizes embedded
//     path segments; the second pass must produce an identical CR.
func FuzzBlockStorageSpecRoundTrip(f *testing.F) {
	// (sizeGB, skuRefResource, skuRefProvider, skuRefRegion, skuRefTenant, skuRefWorkspace, name, provider, tenant, workspace, region)
	f.Add(10, "block-storages/my-bs", "ionos", "de-fra", "t-1", "ws-1", "bs-1", "ionos/de", "t-1", "ws-1", "de-fra")
	f.Add(0, "", "", "", "", "", "", "", "", "", "")
	f.Add(-1, "block-storages/bs", "", "", "", "", "bs", "", "t", "ws", "")
	// length limits
	f.Add(1, "block-storages/"+strings.Repeat("x", 253), "", "", "", "", strings.Repeat("n", 254), "", "t", "", "")
	// provider/region slash/underscore edge cases
	f.Add(1, "block-storages/bs", "///", "de/fra", "t", "ws", "bs", "a/_b", "t", "ws", "de")
	f.Add(1, "block-storages/bs", "ionos/München", "eu/中央", "t", "ws", "bs", "provider/日本語", "t", "ws", "de")

	f.Fuzz(func(t *testing.T,
		sizeGB int,
		skuRefResource, skuRefProvider, skuRefRegion, skuRefTenant, skuRefWorkspace string,
		name, provider, tenant, workspace, region string,
	) {
		domain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name:     name,
					Provider: provider,
				},
				Scope:  kernelresource.Scope{Tenant: tenant, Workspace: workspace},
				Region: region,
			},
			Spec: bsdom.BlockStorageSpec{
				SizeGB: sizeGB,
				SkuRef: commondomain.ReferenceDomain{
					Resource:  skuRefResource,
					Provider:  skuRefProvider,
					Region:    skuRefRegion,
					Tenant:    skuRefTenant,
					Workspace: skuRefWorkspace,
				},
			},
		}

		cr1, err := MapBlockStorageDomainToCR(domain)
		if err != nil {
			return
		}

		domain2, err := MapCRToBlockStorageDomain(cr1)
		if err != nil {
			t.Errorf("CR→domain failed after successful domain→CR: %v", err)
			return
		}

		cr2, err := MapBlockStorageDomainToCR(domain2)
		if err != nil {
			t.Errorf("second domain→CR failed: %v", err)
			return
		}

		domain3, err := MapCRToBlockStorageDomain(cr2)
		if err != nil {
			t.Errorf("second CR→domain failed: %v", err)
			return
		}

		cr3, err := MapBlockStorageDomainToCR(domain3)
		if err != nil {
			t.Errorf("third domain→CR failed: %v", err)
			return
		}

		// SizeGB and CommonMetadata: stable after one round-trip.
		if domain2.Spec.SizeGB != domain3.Spec.SizeGB {
			t.Errorf("SizeGB not stable: %d → %d", domain2.Spec.SizeGB, domain3.Spec.SizeGB)
		}
		if domain2.Name != domain3.Name {
			t.Errorf("Name not stable: %q → %q", domain2.Name, domain3.Name)
		}
		if domain2.Provider != domain3.Provider {
			t.Errorf("Provider not stable: %q → %q", domain2.Provider, domain3.Provider)
		}
		if domain2.Tenant != domain3.Tenant {
			t.Errorf("Tenant not stable: %q → %q", domain2.Tenant, domain3.Tenant)
		}
		if domain2.Workspace != domain3.Workspace {
			t.Errorf("Workspace not stable: %q → %q", domain2.Workspace, domain3.Workspace)
		}
		if domain2.Region != domain3.Region {
			t.Errorf("Region not stable: %q → %q", domain2.Region, domain3.Region)
		}

		// SkuRef domain-level stability: after the first round-trip normalises embedded
		// path segments, domain2 and domain3 must carry identical SkuRef values.
		dref2, dref3 := domain2.Spec.SkuRef, domain3.Spec.SkuRef
		if dref2 != dref3 {
			t.Errorf("SkuRef not stable at domain level:\n  pass1: %+v\n  pass2: %+v", dref2, dref3)
		}

		// SkuRef CR-level stability: cr2 and cr3 must be identical (encoded form).
		bs2 := cr2.(*BlockStorage)
		bs3 := cr3.(*BlockStorage)
		ref2, ref3 := bs2.Spec.SkuRef, bs3.Spec.SkuRef
		if ref2.Provider != ref3.Provider {
			t.Errorf("SkuRef.Provider not stable: %q → %q", ref2.Provider, ref3.Provider)
		}
		if ref2.Region != ref3.Region {
			t.Errorf("SkuRef.Region not stable: %q → %q", ref2.Region, ref3.Region)
		}
		if ref2.Tenant != ref3.Tenant {
			t.Errorf("SkuRef.Tenant not stable: %q → %q", ref2.Tenant, ref3.Tenant)
		}
		if ref2.Workspace != ref3.Workspace {
			t.Errorf("SkuRef.Workspace not stable: %q → %q", ref2.Workspace, ref3.Workspace)
		}
		if ref2.Resource != ref3.Resource {
			t.Errorf("SkuRef.Resource not stable: %q → %q", ref2.Resource, ref3.Resource)
		}
	})
}
