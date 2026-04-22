package kubernetes

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/json"

	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/regional/storage/block-storages/v1"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/regional/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

// FuzzExtractAndStripSegment verifies that extractAndStripSegment never panics on arbitrary input.
func FuzzExtractAndStripSegment(f *testing.F) {
	f.Add("workspaces/ws-1/block-storages/my-storage", "workspaces/")
	f.Add("tenants/t-1/workspaces/ws-1", "workspaces/")
	f.Add("workspaces/ws-1", "workspaces/")
	f.Add("providers/ionos/regions/de-fra", "regions/")
	f.Add("", "workspaces/")
	f.Add("/", "/")
	f.Add("a/b/c", "b/")
	// long paths around Kubernetes' 253-char DNS subdomain limit
	f.Add(strings.Repeat("a", 253)+"/workspaces/ws-1", "workspaces/")
	f.Add(strings.Repeat("a", 254)+"/workspaces/ws-1", "workspaces/")
	f.Add("workspaces/"+strings.Repeat("b", 64), "workspaces/")

	f.Fuzz(func(t *testing.T, resource, segment string) {
		extractAndStripSegment(resource, segment)
	})
}

// FuzzWorkspaceSpecRoundTrip verifies that a workspace spec and its CommonMetadata survive a
// domain→CR→domain→CR round-trip. The invariant is stability: after the first round-trip
// normalizes values, subsequent round-trips produce identical results.
//
// Note: Provider undergoes a "/" ↔ "_" substitution, so the domain value may differ between
// the original and domain2, but domain2 and domain3 must be identical.
func FuzzWorkspaceSpecRoundTrip(f *testing.F) {
	// (specJSON, name, provider, tenant, region)
	f.Add(`{"k":"hello"}`, "ws", "", "t", "")
	f.Add(`{"k":42}`, "ws", "ionos/de", "t", "de-fra")
	f.Add(`{"k":-1}`, "ws", "", "t", "")
	f.Add(`{"k":true}`, "ws", "", "t", "")
	f.Add(`{"k":null}`, "ws", "", "t", "")
	f.Add(`{"k":{"nested":"value"}}`, "ws", "", "t", "")
	f.Add(`{"k":[1,2,3]}`, "ws", "", "t", "")
	f.Add(`{"k":"not-json-value"}`, "ws", "", "t", "")
	f.Add(`{"k e y":"space in key"}`, "ws", "", "t", "")
	f.Add(`{"":"empty key"}`, "ws", "", "t", "")
	// full realistic workspace
	f.Add(`{
		"test-string": "test-value",
		"test-number": 42,
		"test-bool":   true,
		"test-list":   ["a","b","c"],
		"test-map": {
			"inner-string": "inner-value",
			"inner-number": 7,
			"inner-bool":   false,
			"inner-list":   [1,2,3]
		}
	}`, "test-workspace", "ionos/de-fra", "my-tenant", "de-fra")

	// --- Kubernetes length limits ---
	// Names over the DNS subdomain limit (253) and label limit (63)
	f.Add(`{"k":"v"}`, strings.Repeat("a", 254), "", "t", "")
	f.Add(`{"k":"v"}`, strings.Repeat("a", 253), "", "t", "")
	f.Add(`{"k":"v"}`, strings.Repeat("a", 64), "", "t", "")
	// Tenant and region over the label limit (63)
	f.Add(`{"k":"v"}`, "ws", "", strings.Repeat("t", 64), "")
	f.Add(`{"k":"v"}`, "ws", "", "t", strings.Repeat("r", 64))

	// --- Provider slash/underscore edge cases ---
	f.Add(`{"k":"v"}`, "ws", "///", "t", "")                        // only slashes
	f.Add(`{"k":"v"}`, "ws", "___", "t", "")                        // only underscores
	f.Add(`{"k":"v"}`, "ws", "a/b/c/d/e", "t", "")                  // multiple slashes
	f.Add(`{"k":"v"}`, "ws", "/leading", "t", "")                   // leading slash
	f.Add(`{"k":"v"}`, "ws", "trailing/", "t", "")                  // trailing slash
	f.Add(`{"k":"v"}`, "ws", "a/_b", "t", "")                       // mixed slash and underscore
	f.Add(`{"k":"v"}`, "ws", "ionos/München", "t", "")              // non-ASCII
	f.Add(`{"k":"v"}`, "ws", "provider/日本語", "t", "")               // CJK
	f.Add(`{"k":"v"}`, "ws", strings.Repeat("a/", 30)+"b", "t", "") // many slashes

	// --- Deeply nested JSON spec values ---
	f.Add(`{"k":{"a":{"b":{"c":{"d":{"e":{"f":"deep"}}}}}}}`, "ws", "", "t", "")
	f.Add(`{"k":[[[[[[[[[[1]]]]]]]]]]}`, "ws", "", "t", "")
	f.Add(`{"k":{"arr":[[{"x":[[{"y":{"z":0}}]]]}]}}`, "ws", "", "t", "")

	f.Fuzz(func(t *testing.T, specJSON, name, provider, tenant, region string) {
		var spec regional.WorkspaceSpecDomain
		if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
			return // skip inputs that aren't valid JSON objects
		}

		domain := &regional.WorkspaceDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					Name:     name,
					Provider: provider,
				},
				Scope:  scope.Scope{Tenant: tenant},
				Region: region,
			},
			Spec: spec,
		}

		cr1, err := MapWorkspaceDomainToCR(domain)
		if err != nil {
			return
		}

		domain2, err := MapCRToWorkspaceDomain(cr1)
		if err != nil {
			t.Errorf("CR→domain failed after successful domain→CR: %v", err)
			return
		}

		cr2, err := MapWorkspaceDomainToCR(domain2)
		if err != nil {
			t.Errorf("second domain→CR failed: %v", err)
			return
		}

		domain3, err := MapCRToWorkspaceDomain(cr2)
		if err != nil {
			t.Errorf("second CR→domain failed: %v", err)
			return
		}

		// Spec: compare CR specs (map[string]string) for stability
		ws1 := cr1.(*workspacev1.Workspace)
		ws2 := cr2.(*workspacev1.Workspace)
		if len(ws1.Spec) != len(ws2.Spec) {
			t.Errorf("spec length changed after second round-trip: %d → %d", len(ws1.Spec), len(ws2.Spec))
		}
		for k, v1 := range ws1.Spec {
			if v2, ok := ws2.Spec[k]; !ok {
				t.Errorf("spec[%q] lost after second round-trip", k)
			} else if v1 != v2 {
				t.Errorf("spec[%q] not stable: %q → %q", k, v1, v2)
			}
		}

		// Metadata: compare domain2 vs domain3 — provider normalization happens on the
		// first round-trip so domain2 and domain3 must be identical.
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
	})
}

// FuzzBlockStorageSpecRoundTrip verifies that a BlockStorageDomain survives a
// domain→CR→domain→CR→domain→CR round-trip. The interesting complexity is in
// mapDomainReferenceObjectToCR, which extracts "providers/", "regions/",
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
	// embedded path segments in Resource — normalised on first round-trip
	f.Add(10, "providers/ionos/regions/de-fra/tenants/t-1/block-storages/my-bs", "", "", "", "", "bs", "", "t", "ws", "de-fra")
	f.Add(10, "regions/de-fra/workspaces/ws-1/block-storages/bs", "", "", "", "", "bs", "ionos", "t", "", "")
	// adversarial: repeated segment type — requires idempotency fix in mapDomainReferenceObjectToCR
	f.Add(5, "providers/a/providers/b/block-storages/bs", "", "", "", "", "bs", "", "t", "", "")
	f.Add(5, "regions/eu/regions/us/block-storages/bs", "", "", "", "", "bs", "", "t", "", "")
	f.Add(5, "providers/0/providers/0/providers/0", "", "", "", "", "0", "", "0", "", "")
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
		domain := &regional.BlockStorageDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					Name:     name,
					Provider: provider,
				},
				Scope:  scope.Scope{Tenant: tenant, Workspace: workspace},
				Region: region,
			},
			Spec: regional.BlockStorageSpecDomain{
				SizeGB: sizeGB,
				SkuRef: regional.ReferenceObjectDomain{
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
		bs2 := cr2.(*blockstoragev1.BlockStorage)
		bs3 := cr3.(*blockstoragev1.BlockStorage)
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
