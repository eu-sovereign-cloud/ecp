package kubernetes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	rdom "github.com/eu-sovereign-cloud/ecp/resource/region/v1"
	. "github.com/eu-sovereign-cloud/ecp/resource/region/v1/backend/kubernetes"
)

// The happy-path round-trip lives in region_test.go (TestRegionToCR); this file covers the
// rejection and lossy branches of the conversion that nothing else exercises.

func validRegion() *rdom.Region {
	r := &rdom.Region{
		Providers: []rdom.Provider{
			{Name: "seca.network", URL: "https://example.test/network", Version: "v1"},
			{Name: "seca.storage", URL: "https://example.test/storage", Version: "v1"},
		},
		Zones: []rdom.Zone{"zone-a", "zone-b"},
	}
	r.Name = "r1"
	r.ResourceVersion = "42"
	return r
}

func TestRegionFromCR_RejectsIncompleteSpec(t *testing.T) {
	tests := map[string]func(*rdom.Region){
		"no providers":        func(r *rdom.Region) { r.Providers = nil },
		"no zones":            func(r *rdom.Region) { r.Zones = nil },
		"empty provider name": func(r *rdom.Region) { r.Providers[0].Name = "" },
		"empty zone":          func(r *rdom.Region) { r.Zones[0] = "" },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			in := validRegion()
			mutate(in)

			cr, err := RegionToCR(in)
			require.NoError(t, err)

			_, err = RegionFromCR(cr)
			require.Error(t, err)
		})
	}
}

// Regions are platform-managed and RegionToCR writes no labels, so Provider is dropped on
// the way out. Pinned here so a change to that asymmetry is a deliberate one.
func TestRegionToCR_DropsProviderMetadata(t *testing.T) {
	in := validRegion()
	in.Provider = rdom.ProviderID

	cr, err := RegionToCR(in)
	require.NoError(t, err)

	out, err := RegionFromCR(cr)
	require.NoError(t, err)
	require.Empty(t, out.Provider)
}

func TestRegionFromCR_UnsupportedType(t *testing.T) {
	_, err := RegionFromCR(nil)
	require.Error(t, err)
}
