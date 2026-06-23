package kubernetes_test

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/json"

	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1"

	. "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1/backend/kubernetes"
)

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
	f.Add(`{"test-string":"test-value","test-number":42,"test-bool":true}`, "test-workspace", "ionos/de-fra", "my-tenant", "de-fra")

	// Kubernetes length limits
	f.Add(`{"k":"v"}`, strings.Repeat("a", 254), "", "t", "")
	f.Add(`{"k":"v"}`, strings.Repeat("a", 253), "", "t", "")
	f.Add(`{"k":"v"}`, strings.Repeat("a", 64), "", "t", "")
	f.Add(`{"k":"v"}`, "ws", "", strings.Repeat("t", 64), "")
	f.Add(`{"k":"v"}`, "ws", "", "t", strings.Repeat("r", 64))

	// Provider slash/underscore edge cases
	f.Add(`{"k":"v"}`, "ws", "///", "t", "")
	f.Add(`{"k":"v"}`, "ws", "___", "t", "")
	f.Add(`{"k":"v"}`, "ws", "a/b/c/d/e", "t", "")
	f.Add(`{"k":"v"}`, "ws", "/leading", "t", "")
	f.Add(`{"k":"v"}`, "ws", "trailing/", "t", "")
	f.Add(`{"k":"v"}`, "ws", "a/_b", "t", "")
	f.Add(`{"k":"v"}`, "ws", "ionos/München", "t", "")
	f.Add(`{"k":"v"}`, "ws", "provider/nihongo", "t", "")
	f.Add(`{"k":"v"}`, "ws", strings.Repeat("a/", 30)+"b", "t", "")

	// Deeply nested JSON spec values
	f.Add(`{"k":{"a":{"b":{"c":"deep"}}}}`, "ws", "", "t", "")
	f.Add(`{"k":[[[[1]]]]}`, "ws", "", "t", "")

	f.Fuzz(func(t *testing.T, specJSON, name, provider, tenant, region string) {
		var spec wsdom.WorkspaceSpec
		if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
			return // skip inputs that aren't valid JSON objects
		}

		domain := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name:     name,
					Provider: provider,
				},
				Scope:  kernelresource.Scope{Tenant: tenant},
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
		ws1 := cr1.(*Workspace)
		ws2 := cr2.(*Workspace)
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
