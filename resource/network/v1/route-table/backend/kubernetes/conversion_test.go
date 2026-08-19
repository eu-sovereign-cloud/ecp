package kubernetes_test

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
)

const (
	testRouteTableName = "rt1"
	// corruptState is a value no CRD enum allows — what a hand-edited CR looks like.
	corruptState = "halfway"
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
	in.Name = testRouteTableName
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
	in.Name = testRouteTableName

	cr, err := RouteTableToCR(in)
	require.NoError(t, err)

	out, err := RouteTableFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, commondomain.ResourceStatePending, out.Status.Conditions[0].State)
}

func TestRouteTableToCR_UsesNetworkNamespace(t *testing.T) {
	withNetwork := &routetabledom.RouteTable{}
	withNetwork.Name = testRouteTableName
	withNetwork.Tenant = "t1"
	withNetwork.Workspace = "w1"
	withNetwork.Network = "n1"

	cr, err := RouteTableToCR(withNetwork)
	require.NoError(t, err)
	require.NotEmpty(t, cr.GetNamespace())

	other := &routetabledom.RouteTable{}
	other.Name = testRouteTableName
	other.Tenant = "t1"
	other.Workspace = "w1"
	other.Network = "n2"

	otherCR, err := RouteTableToCR(other)
	require.NoError(t, err)
	require.NotEqual(t, cr.GetNamespace(), otherCR.GetNamespace(),
		"different networks must map to different namespaces")
}

// FuzzRouteTableSpecRoundTrip targets the repeated nested spec — a []RouteSpec where each entry
// carries its own Reference. Slice-of-struct specs are where ordering instability and
// nil-vs-empty drift hide, and the per-route TargetRef must come back verbatim once per element.
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

// A route table carries a nested per-route status, so it has two arms that map a resource state.
// Both used to answer "" for a value they did not know — the top-level one made an actively
// broken route table look stateless, the nested one made a single bad route silently vanish.
func TestRouteTableFromCR_RejectsCorruptState(t *testing.T) {
	testCases := []struct {
		name    string
		mutate  func(cr *RouteTable)
		wantMsg string
	}{
		{
			name:    "top-level state",
			mutate:  func(cr *RouteTable) { cr.Status.State = corruptState },
			wantMsg: "route table rt1",
		},
		{
			name:    "nested route state",
			mutate:  func(cr *RouteTable) { cr.Status.Routes[0].State = corruptState },
			wantMsg: "route table rt1",
		},
		{
			name:    "condition state",
			mutate:  func(cr *RouteTable) { cr.Status.Conditions[0].State = corruptState },
			wantMsg: `condition`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cr := activeRouteTableCR(t)
			tc.mutate(cr)

			out, err := RouteTableFromCR(cr)

			require.Error(t, err)
			require.Nil(t, out)
			require.Contains(t, err.Error(), tc.wantMsg)

			var domErr *kernel.Error
			require.ErrorAs(t, err, &domErr, "must stay inspectable across the layer boundary")
			require.Equal(t, kernel.KindValidation, domErr.Kind)
		})
	}
}

// The adapter is the only caller, and it only ever hands over this slice's own type or an
// unstructured one. Anything else is a wiring bug, so it reports as internal rather than as
// something the request could have caused.
func TestRouteTableFromCR_RejectsForeignObject(t *testing.T) {
	out, err := RouteTableFromCR(&corev1.Namespace{})

	require.Error(t, err)
	require.Nil(t, out)

	var domErr *kernel.Error
	require.ErrorAs(t, err, &domErr)
	require.Equal(t, kernel.KindInternal, domErr.Kind)
}

func TestRouteTableToCR_RejectsNil(t *testing.T) {
	out, err := RouteTableToCR(nil)

	require.Error(t, err)
	require.Nil(t, out)

	var domErr *kernel.Error
	require.ErrorAs(t, err, &domErr)
	require.Equal(t, kernel.KindInternal, domErr.Kind)
}

// Converter is what every adapter is now handed instead of the two functions. Crossing the pair
// would compile, so pin that each field is the direction it claims to be.
func TestConverterPairsBothDirections(t *testing.T) {
	in := &routetabledom.RouteTable{}
	in.Name = testRouteTableName
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Network = "n1"

	cr, err := Converter.ToCR(in)
	require.NoError(t, err)
	require.Equal(t, testRouteTableName, cr.GetName())

	out, err := Converter.FromCR(cr)
	require.NoError(t, err)
	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Network, out.Network)
}

// activeRouteTableCR builds a CR whose status is fully populated, so each test only has to
// corrupt the one field it is about.
func activeRouteTableCR(t *testing.T) *RouteTable {
	t.Helper()

	in := &routetabledom.RouteTable{}
	in.Name = testRouteTableName
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Network = "n1"
	in.Status = &routetabledom.RouteTableStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
		Routes: []routetabledom.RouteStatus{{State: commondomain.ResourceStateActive}},
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	obj, err := RouteTableToCR(in)
	require.NoError(t, err)

	cr, ok := obj.(*RouteTable)
	require.True(t, ok)
	require.NotNil(t, cr.Status)
	require.NotEmpty(t, cr.Status.Routes)
	require.NotEmpty(t, cr.Status.Conditions)
	return cr
}

// The write direction used to signal "cannot map this state" with a nil pointer that each caller
// re-reported as a bare string, so the offending value never reached the log.
func TestRouteTableToCR_RejectsUnmappableState(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(rt *routetabledom.RouteTable)
	}{
		{"top-level state", func(rt *routetabledom.RouteTable) { rt.Status.State = corruptState }},
		{"nested route state", func(rt *routetabledom.RouteTable) { rt.Status.Routes[0].State = corruptState }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in := &routetabledom.RouteTable{}
			in.Name = testRouteTableName
			in.Tenant = "t1"
			in.Workspace = "w1"
			in.Network = "n1"
			in.Status = &routetabledom.RouteTableStatus{
				Status: commondomain.Status{State: commondomain.ResourceStateActive},
				Routes: []routetabledom.RouteStatus{{State: commondomain.ResourceStateActive}},
			}
			in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})
			tc.mutate(in)

			out, err := RouteTableToCR(in)

			require.Error(t, err)
			require.Nil(t, out)
			require.Contains(t, err.Error(), "route table rt1", "the error must name the resource it is about")
			require.Contains(t, err.Error(), strconv.Quote(corruptState), "and the value that caused it")

			var domErr *kernel.Error
			require.ErrorAs(t, err, &domErr)
			require.Equal(t, kernel.KindValidation, domErr.Kind)
		})
	}
}
