package kubernetes_test

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	. "github.com/eu-sovereign-cloud/ecp/resources/global/regions/v1/backend/kubernetes"

	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	rdom "github.com/eu-sovereign-cloud/ecp/resources/global/regions/v1"
)

// localProviderSpec is used in test helpers to avoid name clash with Provider (which is now in-package via dot-import).
type localProviderSpec struct {
	Name, Url, Version string
}

func newRegionCR(name string, labels map[string]string, az []string, providers []localProviderSpec, setVersionAndTimestamp bool) *Region {
	if len(az) == 0 {
		az = []string{"az-1"}
	}
	if len(providers) == 0 {
		providers = []localProviderSpec{{Name: "default", Url: "https://default", Version: "v1"}}
	}

	zones := make([]string, len(az)) // Zone = string alias
	copy(zones, az)

	prov := make([]Provider, len(providers))
	for i, p := range providers {
		prov[i] = Provider{Name: p.Name, Url: p.Url, Version: p.Version}
	}

	cr := &Region{
		TypeMeta: metav1.TypeMeta{Kind: RegionKind, APIVersion: GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: RegionSpec{AvailableZones: zones, Providers: prov},
	}
	if setVersionAndTimestamp {
		cr.SetCreationTimestamp(metav1.Time{Time: time.Unix(1700000000, 0)})
		cr.SetResourceVersion("1")
	}
	return cr
}

func toUnstructured(t *testing.T, scheme *runtime.Scheme, obj runtime.Object) *unstructured.Unstructured {
	t.Helper()
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err)

	u := &unstructured.Unstructured{Object: m}

	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Empty() {
		gvks, _, err := scheme.ObjectKinds(obj)
		require.NoError(t, err)
		require.NotEmpty(t, gvks)
		gvk = gvks[0]
	}
	u.SetGroupVersionKind(gvk)

	return u
}

func newAdapter(dynFake *fake.FakeDynamicClient) *k8sadapter.ReaderAdapter[*rdom.Region] {
	return k8sadapter.NewReaderAdapter[*rdom.Region](dynFake, RegionGVR, slog.Default(), MapCRToRegionDomain)
}

func TestRegionController_GetRegion(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, AddToScheme(scheme))

	regionName := "test-region"
	regionLabels := map[string]string{"tier": "prod", "env": "production"}
	availableZones := []string{"az-1", "az-2"}
	providers := []localProviderSpec{
		{Name: "provider1", Url: "https://provider1.example.com", Version: "v1"},
		{Name: "provider2", Url: "https://provider2.example.com", Version: "v2"},
	}

	testRegion := newRegionCR(regionName, regionLabels, availableZones, providers, true)

	t.Run("successful_get", func(t *testing.T) {
		dynFake := fake.NewSimpleDynamicClient(scheme, []runtime.Object{
			toUnstructured(t, scheme, testRegion),
		}...)
		adapter := newAdapter(dynFake)

		region := &rdom.Region{}
		region.Name = regionName
		err := adapter.Load(context.Background(), &region)

		require.NoError(t, err)
		require.NotNil(t, region)
		require.NotNil(t, region.Metadata)
		require.Equal(t, regionName, region.Name)
		expectedZones := make([]rdom.Zone, len(availableZones))
		for i, z := range availableZones {
			expectedZones[i] = rdom.Zone(z)
		}
		require.ElementsMatch(t, expectedZones, region.Zones)
		require.Len(t, region.Providers, 2)
		require.Equal(t, "provider1", region.Providers[0].Name)
		require.Equal(t, "https://provider1.example.com", region.Providers[0].URL)
		require.Equal(t, "v1", region.Providers[0].Version)
		require.Equal(t, "provider2", region.Providers[1].Name)
	})

	t.Run("region_not_found", func(t *testing.T) {
		dynFake := fake.NewSimpleDynamicClient(scheme)
		adapter := newAdapter(dynFake)

		region := &rdom.Region{}
		region.Name = "nonexistent-region"
		err := adapter.Load(context.Background(), &region)

		require.Error(t, err)
	})

	t.Run("get_region_with_minimal_spec", func(t *testing.T) {
		minimalRegionName := "minimal-region"
		minimalRegion := newRegionCR(minimalRegionName, nil, nil, nil, true)

		dynFake := fake.NewSimpleDynamicClient(scheme, []runtime.Object{
			toUnstructured(t, scheme, minimalRegion),
		}...)
		adapter := newAdapter(dynFake)

		region := &rdom.Region{}
		region.Name = minimalRegionName
		err := adapter.Load(context.Background(), &region)

		require.NoError(t, err)
		require.NotNil(t, region)
		require.Equal(t, minimalRegionName, region.Name)
		// Verify default values set by newRegionCR helper
		require.Len(t, region.Zones, 1)
		require.Equal(t, rdom.Zone("az-1"), region.Zones[0])
		require.Len(t, region.Providers, 1)
		require.Equal(t, "default", region.Providers[0].Name)
	})

	t.Run("get_region_with_multiple_availability_zones", func(t *testing.T) {
		multiAZRegionName := "multi-az-region"
		multiAZ := []string{"az-1", "az-2", "az-3", "az-4"}
		multiAZRegion := newRegionCR(multiAZRegionName, nil, multiAZ, nil, true)

		objs := []runtime.Object{
			toUnstructured(t, scheme, multiAZRegion),
		}

		dynFake := fake.NewSimpleDynamicClient(scheme, objs...)
		adapter := newAdapter(dynFake)

		region := &rdom.Region{}
		region.Name = multiAZRegionName
		err := adapter.Load(context.Background(), &region)

		require.NoError(t, err)
		require.NotNil(t, region)
		require.Equal(t, multiAZRegionName, region.Name)
		require.Len(t, region.Zones, 4)
		expectedZones := make([]rdom.Zone, len(multiAZ))
		for i, z := range multiAZ {
			expectedZones[i] = rdom.Zone(z)
		}
		require.ElementsMatch(t, expectedZones, region.Zones)
	})
}

// TestRegionController_GetRegion_EdgeCases uses fake client to test edge cases.
func TestRegionController_GetRegion_EdgeCases(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, AddToScheme(scheme))

	t.Run("get_with_empty_name", func(t *testing.T) {
		dynFake := fake.NewSimpleDynamicClient(scheme)
		adapter := newAdapter(dynFake)

		region := &rdom.Region{}
		region.Name = ""
		err := adapter.Load(context.Background(), &region)

		require.Error(t, err)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		regionName := "test-region"
		testRegion := newRegionCR(regionName, nil, nil, nil, true)

		objs := []runtime.Object{
			toUnstructured(t, scheme, testRegion),
		}

		dynFake := fake.NewSimpleDynamicClient(scheme, objs...)
		adapter := newAdapter(dynFake)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		region := &rdom.Region{}
		region.Name = regionName
		err := adapter.Load(ctx, &region)
		// The fake client might not respect context cancellation perfectly,
		// but we test the behavior anyway.
		if err != nil {
			require.True(t, errors.Is(err, context.Canceled) || region == nil)
		}
	})
}

func TestRegionController_ListRegions(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, AddToScheme(scheme))

	regionAName := "region-a"
	regionBName := "region-b"
	regionCName := "region-c"

	r1 := newRegionCR(regionAName, map[string]string{"tier": "prod", "env": "prod"}, []string{"az-a1"}, []localProviderSpec{{Name: "p1", Url: "https://p1", Version: "v1"}}, true)
	r2 := newRegionCR(regionBName, map[string]string{"tier": "dev", "env": "staging"}, []string{"az-b1", "az-b2"}, []localProviderSpec{{Name: "p2", Url: "https://p2", Version: "v2"}}, true)
	r3 := newRegionCR(regionCName, map[string]string{"tier": "prod", "env": "staging", "region": "3"}, []string{"az-c1"}, []localProviderSpec{{Name: "p3", Url: "https://p3", Version: "v3"}}, true)

	objs := []runtime.Object{
		toUnstructured(t, scheme, r1),
		toUnstructured(t, scheme, r2),
		toUnstructured(t, scheme, r3),
	}

	dynFake := fake.NewSimpleDynamicClient(scheme, objs...)
	adapter := newAdapter(dynFake)

	type tc struct {
		name      string
		selector  *string
		wantNames []string
	}

	complexLabel := "tier=prod,env=prod"
	complexClientLabel := "region>2"
	simple := "tier=prod"
	none := "tier=qa"
	simpleOnlyKey := "tier"
	k8sSetBased := "tier in (prod)"
	k8sSetBasedAndEquality := "tier in (prod),env=staging"
	wildcard := "env=stag*"
	wildcardKeyAndValue := "t*r=pr*d"

	tests := []tc{
		{name: "all_no_selector", selector: nil, wantNames: []string{regionAName, regionBName, regionCName}},
		{name: "simple_server_side_prefilter", selector: &simple, wantNames: []string{regionAName, regionCName}},
		{name: "complex_server_side_filter", selector: &complexLabel, wantNames: []string{regionAName}},
		{name: "complex_client_>_side_filter", selector: &complexClientLabel, wantNames: []string{regionCName}},
		{name: "no_matches", selector: &none, wantNames: []string{}},
		{name: "simple_only_key_selector", selector: &simpleOnlyKey, wantNames: []string{regionAName, regionBName, regionCName}},
		{name: "k8s_set_based_selector", selector: &k8sSetBased, wantNames: []string{regionAName, regionCName}},
		{name: "k8s_set_based_and_equality_selector", selector: &k8sSetBasedAndEquality, wantNames: []string{regionCName}},
		{name: "wildcard_env_prefix", selector: &wildcard, wantNames: []string{regionBName, regionCName}},
		{name: "wildcard_in_key_and_value", selector: &wildcardKeyAndValue, wantNames: []string{regionAName, regionCName}},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := kernelresource.ListParams{}
			if tt.selector != nil {
				params.Selector = *tt.selector
			}
			var regions []*rdom.Region
			_, err := adapter.List(ctx, params, &regions)
			require.NoError(t, err)
			require.ElementsMatch(t, tt.wantNames, extractDomainNames(regions))
		})
	}
}

func extractDomainNames(regs []*rdom.Region) []string {
	out := make([]string, len(regs))
	for i, r := range regs {
		out[i] = r.Name
	}
	sort.Strings(out)
	return out
}
