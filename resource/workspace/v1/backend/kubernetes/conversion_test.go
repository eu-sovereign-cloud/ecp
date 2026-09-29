package kubernetes_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"

	. "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

// FuzzWorkspaceSpecRoundTrip verifies that a workspace spec and its CommonMetadata survive a
// domain→CR→domain→CR round-trip. The invariant is stability: after the first round-trip
// normalizes values, subsequent round-trips produce identical results.
//
// Note: Provider undergoes a "/" ↔ "_" substitution, so the domain value may differ between
// the original and domain2, but domain2 and domain3 must be identical.
func FuzzWorkspaceSpecRoundTrip(f *testing.F) {
	// (specJSON, name, provider, tenant, region)
	f.Add(`{"k":"hello"}`, "ws", "", "t", "") // no region: rejected
	f.Add(`{"k":"hello"}`, "ws", "", "t", "r")
	f.Add(`{"k":42}`, "ws", "ionos/de", "t", "de-fra")
	f.Add(`{"k":-1}`, "ws", "", "t", "r")
	f.Add(`{"k":true}`, "ws", "", "t", "r")
	f.Add(`{"k":null}`, "ws", "", "t", "r")
	f.Add(`{"k":{"nested":"value"}}`, "ws", "", "t", "r")
	f.Add(`{"k":[1,2,3]}`, "ws", "", "t", "r")
	f.Add(`{"k":"not-json-value"}`, "ws", "", "t", "r")
	f.Add(`{"k e y":"space in key"}`, "ws", "", "t", "r")
	f.Add(`{"":"empty key"}`, "ws", "", "t", "r")
	// full realistic workspace
	f.Add(`{"test-string":"test-value","test-number":42,"test-bool":true}`, "test-workspace", "ionos/de-fra", "my-tenant", "de-fra")

	// Kubernetes length limits
	f.Add(`{"k":"v"}`, strings.Repeat("a", 254), "", "t", "r")
	f.Add(`{"k":"v"}`, strings.Repeat("a", 253), "", "t", "r")
	f.Add(`{"k":"v"}`, strings.Repeat("a", 64), "", "t", "r")
	f.Add(`{"k":"v"}`, "ws", "", strings.Repeat("t", 64), "r")
	f.Add(`{"k":"v"}`, "ws", "", "t", strings.Repeat("r", 64))

	// Provider slash/underscore edge cases
	f.Add(`{"k":"v"}`, "ws", "///", "t", "r")
	f.Add(`{"k":"v"}`, "ws", "___", "t", "r")
	f.Add(`{"k":"v"}`, "ws", "a/b/c/d/e", "t", "r")
	f.Add(`{"k":"v"}`, "ws", "/leading", "t", "r")
	f.Add(`{"k":"v"}`, "ws", "trailing/", "t", "r")
	f.Add(`{"k":"v"}`, "ws", "a/_b", "t", "r")
	f.Add(`{"k":"v"}`, "ws", "ionos/München", "t", "r")
	f.Add(`{"k":"v"}`, "ws", "provider/nihongo", "t", "r")
	f.Add(`{"k":"v"}`, "ws", strings.Repeat("a/", 30)+"b", "t", "r")

	// Deeply nested JSON spec values
	f.Add(`{"k":{"a":{"b":{"c":"deep"}}}}`, "ws", "", "t", "r")
	f.Add(`{"k":[[[[1]]]]}`, "ws", "", "t", "r")

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

		cr1, err := WorkspaceToCR(domain)
		if region == "" {
			// A workspace is keyed by region: without one it must not be written anywhere.
			if err == nil {
				t.Errorf("domain→CR accepted a workspace with no region, placing it in %q", cr1.GetNamespace())
			}
			return
		}
		if err != nil {
			return
		}
		requireRegionInSync(t, cr1, region)

		domain2, err := WorkspaceFromCR(cr1)
		if err != nil {
			t.Errorf("CR→domain failed after successful domain→CR: %v", err)
			return
		}

		cr2, err := WorkspaceToCR(domain2)
		if err != nil {
			t.Errorf("second domain→CR failed: %v", err)
			return
		}

		domain3, err := WorkspaceFromCR(cr2)
		if err != nil {
			t.Errorf("second CR→domain failed: %v", err)
			return
		}
		requireRegionInSync(t, cr2, region)

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
		if domain2.Region != region || domain3.Region != region {
			t.Errorf("Region not preserved: %q → %q → %q", region, domain2.Region, domain3.Region)
		}
	})
}

// requireRegionInSync fails unless cr carries region both as its field and as its internal
// region label. The field is what the plugin provisions into and the namespace is derived from;
// the label is what the gateway's list filter and the delegator's region scope select on, so a
// CR on which they disagree is listed, and reconciled, as one region and created in another.
func requireRegionInSync(t *testing.T, cr client.Object, region string) {
	t.Helper()

	ws, ok := cr.(*Workspace)
	if !ok {
		t.Fatalf("WorkspaceToCR returned %T, want *Workspace", cr)
	}
	if ws.Region != region {
		t.Errorf("CR region field = %q, want %q", ws.Region, region)
	}
	if got := ws.GetLabels()[k8slabels.InternalRegionLabel]; got != region {
		t.Errorf("CR region label = %q, want %q (the field is %q)", got, region, ws.Region)
	}
}

// newWorkspace builds a minimal domain workspace for the region-placement tests below.
func newWorkspace(tenant, name, region string) *wsdom.Workspace {
	return &wsdom.Workspace{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: name},
			Scope:          kernelresource.Scope{Tenant: tenant},
			Region:         region,
		},
	}
}

// TestWorkspaceToCR_RegionPlacement is the uniqueness guarantee of issue #396: a workspace is
// identified by tenant *and* region, so the same name in two regions is two CRs in two
// namespaces rather than one that the second write overwrites.
func TestWorkspaceToCR_RegionPlacement(t *testing.T) {
	one, err := WorkspaceToCR(newWorkspace("t1", "ws1", "region-one"))
	require.NoError(t, err)
	two, err := WorkspaceToCR(newWorkspace("t1", "ws1", "region-two"))
	require.NoError(t, err)

	require.Equal(t, one.GetName(), two.GetName(), "the CR keeps the name the caller asked for")
	require.NotEqual(t, one.GetNamespace(), two.GetNamespace(),
		"the same workspace name in two regions must not resolve to one CR")
	require.Equal(t, k8sadapter.ComputeRegionNamespace(newWorkspace("t1", "ws1", "region-one")), one.GetNamespace())

	require.Equal(t, "region-one", one.(*Workspace).Region, "region is a field on the CR, not only a label")
	require.Equal(t, "region-one", one.GetLabels()[k8slabels.InternalRegionLabel],
		"the label is still written: it is what the gateway's list filter selects on")

	// A workspace with no region has no namespace of its own, so it is refused rather than
	// placed in the plain tenant namespace, where no request addressed to a region finds it.
	_, err = WorkspaceToCR(newWorkspace("t1", "ws1", ""))
	require.Error(t, err)
	require.Equal(t, kernel.KindValidation, kernel.AsError(err).Kind)
}

// TestWorkspaceFromCR_RegionFromField pins the field as the one source of the region: a CR's
// label is never consulted, so a CR that carries only the label (hand-written, or from before
// the field existed) reads as region-less instead of being quietly adopted, and one on which the
// two disagree reads as the region its namespace was derived from.
func TestWorkspaceFromCR_RegionFromField(t *testing.T) {
	tests := []struct {
		name       string
		field      string
		label      string
		wantRegion string
	}{
		{name: "field and label agree", field: "region-one", label: "region-one", wantRegion: "region-one"},
		{name: "the field wins over a diverged label", field: "region-one", label: "region-two", wantRegion: "region-one"},
		{name: "a label alone is not a region", label: "region-one", wantRegion: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cr := &Workspace{Region: tc.field}
			cr.SetName("ws1")
			cr.SetLabels(map[string]string{
				k8slabels.InternalTenantLabel: "t1",
				k8slabels.InternalRegionLabel: tc.label,
			})

			typed, err := WorkspaceFromCR(cr)
			require.NoError(t, err)
			require.Equal(t, tc.wantRegion, typed.Region)

			// The dynamic client hands the reader unstructured objects: same answer.
			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cr)
			require.NoError(t, err)
			untyped, err := WorkspaceFromCR(&unstructured.Unstructured{Object: raw})
			require.NoError(t, err)
			require.Equal(t, tc.wantRegion, untyped.Region)
		})
	}
}
