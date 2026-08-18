package kubernetes_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
)

const n1 = "n1"

func TestNetworkConversionRoundTrip(t *testing.T) {
	in := &netdom.Network{
		Spec: netdom.NetworkSpec{
			CIDR:            netdom.CIDR{IPv4: "10.0.0.0/16", IPv6: "fd00::/48"},
			AdditionalCIDRs: []netdom.CIDR{{IPv4: "10.1.0.0/16"}, {IPv6: "fd01::/48"}},
			SkuRef:          commondomain.Reference{Resource: "network-skus/sku1"},
		},
	}
	in.Name = n1
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Provider = netdom.ProviderID
	in.Region = "r1"
	in.Labels = map[string]string{"env": "prod"}
	in.Annotations = map[string]string{"note": "primary"}
	in.Status = &netdom.NetworkStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := NetworkToCR(in)
	require.NoError(t, err)

	out, err := NetworkFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Region, out.Region)
	require.Equal(t, in.Provider, out.Provider)
	require.Equal(t, in.Labels, out.Labels)
	require.Equal(t, in.Annotations, out.Annotations)
	require.Equal(t, in.Spec.CIDR, out.Spec.CIDR)
	require.Equal(t, in.Spec.AdditionalCIDRs, out.Spec.AdditionalCIDRs)
	require.Equal(t, in.Spec.SkuRef.Resource, out.Spec.SkuRef.Resource)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
}

func TestNetworkToCR_DefaultPendingCondition(t *testing.T) {
	in := &netdom.Network{
		Spec: netdom.NetworkSpec{CIDR: netdom.CIDR{IPv4: "10.0.0.0/16"}},
	}
	in.Name = n1

	cr, err := NetworkToCR(in)
	require.NoError(t, err)

	out, err := NetworkFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, commondomain.ResourceStatePending, out.Status.Conditions[0].State)
}

// Provider carries a "/" that is stored as "_" in the CR label, so it must survive the
// substitution both ways.
func TestNetworkConversion_ProviderSlashSurvives(t *testing.T) {
	in := &netdom.Network{}
	in.Name = n1
	in.Provider = "ionos/de"

	cr, err := NetworkToCR(in)
	require.NoError(t, err)
	require.NotContains(t, cr.GetLabels(), "/", "provider label must not contain a slash")

	out, err := NetworkFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, "ionos/de", out.Provider)
}

func TestNetworkConversion_SkuRefOptional(t *testing.T) {
	in := &netdom.Network{
		Spec: netdom.NetworkSpec{CIDR: netdom.CIDR{IPv4: "10.0.0.0/16"}},
	}
	in.Name = n1

	cr, err := NetworkToCR(in)
	require.NoError(t, err)

	out, err := NetworkFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, commondomain.Reference{}, out.Spec.SkuRef)
	require.Empty(t, out.Spec.AdditionalCIDRs)
}

func TestNetworkToCR_NilNetwork(t *testing.T) {
	_, err := NetworkToCR(nil)
	require.Error(t, err)
}

// TestNetworkToCR_LabelKeysAreSorted pins the ordering of commonData.labels, which every *ToCR in
// every slice builds the same way and which is not cosmetic.
//
// The key list comes from a Go map, whose iteration order is deliberately randomised. The writer
// adapter compares stored commonData against desired to decide whether an update needs to write at
// all (see framework/backend/kubernetes/adapter.go), and an equal-but-reordered list compares
// unequal - so an unsorted list makes a no-op PUT rewrite the CR, bump its resourceVersion, and
// trigger a reconcile that hands the plugin a level-triggered Update for a request that changed
// nothing. Nine keys, so an unsorted build has a 1-in-9! chance of passing by luck.
func TestNetworkToCR_LabelKeysAreSorted(t *testing.T) {
	in := &netdom.Network{Spec: netdom.NetworkSpec{CIDR: netdom.CIDR{IPv4: "10.0.0.0/16"}}}
	in.Name = n1
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Labels = map[string]string{
		"team": "platform", "env": "prod", "tier": "frontend", "zone": "a", "app": "api",
		"owner": "core", "stage": "live", "region": "eu", "billing": "cc1",
	}

	cr, err := NetworkToCR(in)
	require.NoError(t, err)

	keys := cr.(*Network).CommonData.Labels
	require.True(t, slices.IsSorted(keys), "commonData.labels must be sorted, got %v", keys)
	require.ElementsMatch(t, slices.Collect(maps.Keys(in.Labels)), keys, "every label key must be present")
}

// FuzzNetworkSpecRoundTrip fuzzes the label and CIDR input shapes, which the existing global and
// workspace-scoped fuzz targets do not reach. The invariant is stability, not identity: the first
// round-trip normalizes (provider "/"↔"_", label keying), so domain2 and domain3 must match.
func FuzzNetworkSpecRoundTrip(f *testing.F) {
	// (name, provider, tenant, workspace, region, ipv4, ipv6, labelKey, labelValue)
	f.Add("n", "", "t", "w", "", "10.0.0.0/16", "", "env", "prod")
	f.Add("net", "ionos/de", "t", "w", "de-fra", "10.0.0.0/16", "fd00::/48", "env", "prod")
	f.Add("n", "a/b/c/d", "t", "w", "r", "", "", "k", "v")
	f.Add("n", "___", "t", "w", "r", "0.0.0.0/0", "::/0", "k", "v")
	f.Add("n", "/leading", "t", "w", "r", "10.0.0.0/16", "", "", "")
	f.Add("n", "trailing/", "t", "w", "r", "10.0.0.0/16", "", "a.b/c", "v")
	// label keys that collide with the internal label namespace
	f.Add("n", "", "t", "w", "r", "10.0.0.0/16", "", "ecp.internal/tenant", "spoofed")
	f.Add("n", "", "t", "w", "r", "10.0.0.0/16", "", strings.Repeat("k", 64), "v")
	f.Add(strings.Repeat("n", 253), "", "t", "w", "r", "10.0.0.0/16", "", "k", "v")
	f.Add("n", "", "t", "w", "r", "not-a-cidr", "also-not", "k", "v")

	f.Fuzz(func(t *testing.T, name, provider, tenant, workspace, region, ipv4, ipv6, labelKey, labelValue string) {
		domain := &netdom.Network{
			Spec: netdom.NetworkSpec{CIDR: netdom.CIDR{IPv4: ipv4, IPv6: ipv6}},
		}
		domain.Name = name
		domain.Provider = provider
		domain.Tenant = tenant
		domain.Workspace = workspace
		domain.Region = region
		if labelKey != "" {
			domain.Labels = map[string]string{labelKey: labelValue}
		}

		cr1, err := NetworkToCR(domain)
		if err != nil {
			return
		}

		domain2, err := NetworkFromCR(cr1)
		if err != nil {
			t.Errorf("CR→domain failed after successful domain→CR: %v", err)
			return
		}

		cr2, err := NetworkToCR(domain2)
		if err != nil {
			t.Errorf("second domain→CR failed: %v", err)
			return
		}

		domain3, err := NetworkFromCR(cr2)
		if err != nil {
			t.Errorf("second CR→domain failed: %v", err)
			return
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
		if domain2.Spec.CIDR != domain3.Spec.CIDR {
			t.Errorf("CIDR not stable: %+v → %+v", domain2.Spec.CIDR, domain3.Spec.CIDR)
		}
		if len(domain2.Labels) != len(domain3.Labels) {
			t.Errorf("label count not stable: %d → %d", len(domain2.Labels), len(domain3.Labels))
		}
		for k, v2 := range domain2.Labels {
			if v3, ok := domain3.Labels[k]; !ok {
				t.Errorf("label %q lost on second round-trip", k)
			} else if v2 != v3 {
				t.Errorf("label %q not stable: %q → %q", k, v2, v3)
			}
		}
		if cr1.GetNamespace() != cr2.GetNamespace() {
			t.Errorf("namespace not stable: %q → %q", cr1.GetNamespace(), cr2.GetNamespace())
		}
	})
}

func TestNetworkFromCR_UnsupportedType(t *testing.T) {
	_, err := NetworkFromCR(nil)
	require.Error(t, err)
}
