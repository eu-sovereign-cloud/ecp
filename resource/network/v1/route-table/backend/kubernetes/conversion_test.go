package kubernetes_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
)

func TestRouteTableConversionRoundTrip(t *testing.T) {
	in := &routetabledom.RouteTable{
		Spec: routetabledom.RouteTableSpec{
			Routes: []routetabledom.RouteSpec{
				{
					DestinationCidrBlock: "10.0.0.0/24",
					// Reference.resource: {collection}/{name}
					// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
					TargetRef: commondomain.Reference{Resource: "instances/inst1"},
				},
			},
		},
	}
	in.Name = "rt1"
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Network = "n1"
	in.Provider = routetabledom.ProviderID
	in.Region = "r1"
	in.Status = &routetabledom.RouteTableStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
		Routes: []routetabledom.RouteStatus{
			{State: commondomain.ResourceStateActive},
		},
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := RouteTableToCR(in)
	require.NoError(t, err)

	out, err := RouteTableFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Network, out.Network)
	require.Equal(t, in.Region, out.Region)
	require.Len(t, out.Spec.Routes, 1)
	require.Equal(t, "10.0.0.0/24", out.Spec.Routes[0].DestinationCidrBlock)
	require.Equal(t, "instances/inst1", out.Spec.Routes[0].TargetRef.Resource)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
	require.Len(t, out.Status.Routes, 1)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.Routes[0].State)
}

func TestRouteTableToCR_DefaultPendingCondition(t *testing.T) {
	in := &routetabledom.RouteTable{
		Spec: routetabledom.RouteTableSpec{
			Routes: []routetabledom.RouteSpec{{DestinationCidrBlock: "10.0.0.0/24"}},
		},
	}
	in.Name = "rt1"

	cr, err := RouteTableToCR(in)
	require.NoError(t, err)

	out, err := RouteTableFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, commondomain.ResourceStatePending, out.Status.Conditions[0].State)
}

func TestRouteTableToCR_UsesNetworkNamespace(t *testing.T) {
	withNetwork := &routetabledom.RouteTable{}
	withNetwork.Name = "rt1"
	withNetwork.Tenant = "t1"
	withNetwork.Workspace = "w1"
	withNetwork.Network = "n1"

	cr, err := RouteTableToCR(withNetwork)
	require.NoError(t, err)
	require.NotEmpty(t, cr.GetNamespace())

	other := &routetabledom.RouteTable{}
	other.Name = "rt1"
	other.Tenant = "t1"
	other.Workspace = "w1"
	other.Network = "n2"

	otherCR, err := RouteTableToCR(other)
	require.NoError(t, err)
	require.NotEqual(t, cr.GetNamespace(), otherCR.GetNamespace(),
		"different networks must map to different namespaces")
}

// isWellFormedRefPath reports whether a reference path is shaped like a SECA resource path, for
// which commonbackend's scope embed/extract pair is a true inverse.
//
// ponytail: known ceiling, mirrored from the security-group-rule fuzz target.
// extractAndStripSegment strips a scope anchor from wherever it appears while
// embedScopeInResource always re-inserts it before the final two segments, so the pair only
// round-trips for paths with at most one "tenants/" and one "workspaces/" anchor and no empty
// segments. Real SECA references are always canonical; widen this if the helpers are reworked.
func isWellFormedRefPath(p string) bool {
	return !strings.Contains(p, "//") &&
		strings.Count(p, "tenants") <= 1 &&
		strings.Count(p, "workspaces") <= 1
}

// FuzzRouteTableSpecRoundTrip targets the repeated nested spec — a []RouteSpec where each entry
// carries its own Reference. Slice-of-struct specs are where ordering instability and
// nil-vs-empty drift hide, and the per-route TargetRef exercises the same scope embed/extract
// path as a standalone reference but once per element.
//
// The invariant is stability, not identity: the first round-trip normalizes, so domain2 and
// domain3 must be identical.
func FuzzRouteTableSpecRoundTrip(f *testing.F) {
	// (name, provider, network, dest1, target1, dest2, target2, routeCount)
	f.Add("rt", "", "n1", "10.0.0.0/24", "instances/i1", "0.0.0.0/0", "internet-gateways/igw1", 2)
	f.Add("rt", "ionos/de", "n1", "10.0.0.0/24", "instances/i1", "", "", 1)
	f.Add("rt", "", "n1", "", "", "", "", 0)
	f.Add("rt", "a/b/c", "n1", "::/0", "tenants/t/instances/i1", "10.0.0.0/8", "public-ips/ip1", 2)
	f.Add("rt", "", "n1", "not-a-cidr", "seca.network/v1/tenants/t/workspaces/w/instances/i1", "", "", 1)
	// routeCount out of range must not panic the conversion
	f.Add("rt", "", "n1", "10.0.0.0/24", "instances/i1", "10.1.0.0/24", "instances/i2", 99)
	f.Add("rt", "", "n1", "10.0.0.0/24", "instances/i1", "10.1.0.0/24", "instances/i2", -1)

	f.Fuzz(func(t *testing.T, name, provider, network, dest1, target1, dest2, target2 string, routeCount int) {
		if !isWellFormedRefPath(target1) || !isWellFormedRefPath(target2) {
			return
		}

		all := []routetabledom.RouteSpec{
			{DestinationCidrBlock: dest1, TargetRef: commondomain.Reference{Resource: target1}},
			{DestinationCidrBlock: dest2, TargetRef: commondomain.Reference{Resource: target2}},
		}
		n := min(max(routeCount, 0), len(all))

		domain := &routetabledom.RouteTable{
			Spec: routetabledom.RouteTableSpec{Routes: all[:n]},
		}
		domain.Name = name
		domain.Provider = provider
		domain.Tenant = "t"
		domain.Workspace = "w"
		domain.Network = network

		cr1, err := RouteTableToCR(domain)
		if err != nil {
			return
		}

		domain2, err := RouteTableFromCR(cr1)
		if err != nil {
			t.Errorf("CR→domain failed after successful domain→CR: %v", err)
			return
		}

		cr2, err := RouteTableToCR(domain2)
		if err != nil {
			t.Errorf("second domain→CR failed: %v", err)
			return
		}

		domain3, err := RouteTableFromCR(cr2)
		if err != nil {
			t.Errorf("second CR→domain failed: %v", err)
			return
		}

		// Route count must survive: the conversion allocates with make([]T, len(src)) on both
		// sides, so a length change means an element was dropped or duplicated.
		if len(domain2.Spec.Routes) != len(domain3.Spec.Routes) {
			t.Errorf("route count not stable: %d → %d", len(domain2.Spec.Routes), len(domain3.Spec.Routes))
			return
		}
		if len(domain2.Spec.Routes) != n {
			t.Errorf("route count changed on first round-trip: %d → %d", n, len(domain2.Spec.Routes))
			return
		}
		// Order matters: routes are positional, so element i must stay element i.
		for i := range domain2.Spec.Routes {
			if !reflect.DeepEqual(domain2.Spec.Routes[i], domain3.Spec.Routes[i]) {
				t.Errorf("route %d not stable: %+v → %+v", i, domain2.Spec.Routes[i], domain3.Spec.Routes[i])
			}
		}
		if domain2.Name != domain3.Name {
			t.Errorf("Name not stable: %q → %q", domain2.Name, domain3.Name)
		}
		if domain2.Provider != domain3.Provider {
			t.Errorf("Provider not stable: %q → %q", domain2.Provider, domain3.Provider)
		}
		if domain2.Network != domain3.Network {
			t.Errorf("Network not stable: %q → %q", domain2.Network, domain3.Network)
		}
		if cr1.GetNamespace() != cr2.GetNamespace() {
			t.Errorf("namespace not stable: %q → %q", cr1.GetNamespace(), cr2.GetNamespace())
		}
	})
}
