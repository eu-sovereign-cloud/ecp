package globalprovider

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	sdkregion "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	generatedv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/kubeclient"
)

func TestRegionController_ListRegions(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, regionsv1.AddToScheme(scheme))
	regionAName := "region-a"
	regionBName := "region-b"
	regionCName := "region-c"

	r1 := newRegionCR(regionAName, map[string]string{"tier": "prod", "env": "prod"}, []string{"az-a1"}, []providerSpec{{Name: "p1", Url: "https://p1", Version: "v1"}}, true)
	r2 := newRegionCR(regionBName, map[string]string{"tier": "dev", "env": "staging"}, []string{"az-b1", "az-b2"}, []providerSpec{{Name: "p2", Url: "https://p2", Version: "v2"}}, true)
	r3 := newRegionCR(regionCName, map[string]string{"tier": "prod", "env": "staging", "region": "3"}, []string{"az-c1"}, []providerSpec{{Name: "p3", Url: "https://p3", Version: "v3"}}, true)

	objs := []runtime.Object{
		toUnstructured(t, scheme, r1),
		toUnstructured(t, scheme, r2),
		toUnstructured(t, scheme, r3),
	}

	dyn := fake.NewSimpleDynamicClient(scheme, objs...)

	rc := &RegionController{
		client: &kubeclient.KubeClient{Client: dyn},
		logger: slog.Default(),
	}

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
	invalid := "tier=+=prod" // Intentionally invalid ('=+='); filter.MatchLabels should error -> all skipped.
	wildcard := "env=stag*"
	wildcardKeyAndValue := "t*r=pr*d"

	tests := []tc{
		{
			name:      "all_no_selector",
			selector:  nil,
			wantNames: []string{regionAName, regionBName, regionCName},
		},
		{
			name:      "simple_server_side_prefilter",
			selector:  &simple,
			wantNames: []string{regionAName, regionCName},
		},
		{
			name:      "complex_server_side_filter",
			selector:  &complexLabel,
			wantNames: []string{regionAName},
		},
		{
			name:      "complex_client_>_side_filter",
			selector:  &complexClientLabel,
			wantNames: []string{regionCName},
		},
		{
			name:      "no_matches",
			selector:  &none,
			wantNames: []string{},
		},
		{
			name:      "invalid_selector_skips_all",
			selector:  &invalid,
			wantNames: []string{},
		},
		{
			name:      "simple_only_key_selector",
			selector:  &simpleOnlyKey,
			wantNames: []string{regionAName, regionBName, regionCName},
		},
		{
			name:      "k8s_set_based_selector",
			selector:  &k8sSetBased,
			wantNames: []string{regionAName, regionCName},
		},
		{
			name:      "k8s_set_based_and_equality_selector",
			selector:  &k8sSetBasedAndEquality,
			wantNames: []string{regionCName},
		},
		{
			name:      "wildcard_env_prefix",
			selector:  &wildcard,
			wantNames: []string{regionBName, regionCName},
		},
		{
			name:      "wildcard_in_key_and_value",
			selector:  &wildcardKeyAndValue,
			wantNames: []string{regionAName, regionCName},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter, err := rc.ListRegions(ctx, sdkregion.ListRegionsParams{Labels: tt.selector})
			require.NoError(t, err)
			regions := iter.Items

			require.NoError(t, err)
			require.ElementsMatch(t, tt.wantNames, extractNames(regions))
		})
	}
}

func TestMain(m *testing.M) {
	// Resolve CRD directory path relative to this test file's package directory.
	wd, _ := os.Getwd()
	crdDir := filepath.Clean(filepath.Join(wd, "../../../../api/generated/crds/regions"))

	// Ensure envtest downloads the required control-plane binaries and installs the CRDs.
	testenv := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{crdDir},
		DownloadBinaryAssets:  true,
		BinaryAssetsDirectory: filepath.Join(os.TempDir(), "envtest-binaries"),
	}
	var err error
	cfg, err = testenv.Start()
	if err != nil {
		slog.Error("failed to start test environment", "error", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := testenv.Stop(); err != nil {
		slog.Error("failed to stop test environment", "error", err)
	}
	os.Exit(code)
}

var cfg *rest.Config

// TestRegionController_ListRegions_Pagination - uses envtest, as fake kube does not support limit
func TestRegionController_ListRegions_Pagination(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, regionsv1.AddToScheme(scheme))

	dynClient, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)

	rc := &RegionController{
		client: &kubeclient.KubeClient{Client: dynClient},
		logger: slog.Default(),
	}

	// --- Create test data ---
	totalRegions := 5
	regionNames := make([]string, totalRegions)
	for i := 0; i < totalRegions; i++ {
		name := fmt.Sprintf("region-%c", 'a'+i)
		regionNames[i] = name
		cr := newRegionCR(name, nil, nil, nil, false)
		unstructuredCR := toUnstructured(t, scheme, cr)

		_, err := dynClient.Resource(regionsv1.GroupVersionResource).Create(context.Background(), unstructuredCR, metav1.CreateOptions{})
		require.NoError(t, err)
	}
	sort.Strings(regionNames)

	ctx := context.Background()
	limit := 2

	// 1. First page: limit=2, no skip token
	var iter *sdkregion.RegionIterator
	t.Run("first_page", func(t *testing.T) {
		iter, err = rc.ListRegions(ctx, sdkregion.ListRegionsParams{Limit: &limit})
		require.NoError(t, err)
		require.NotNil(t, iter)

		require.Len(t, iter.Items, 2)
		require.ElementsMatch(t, regionNames[0:2], extractNames(iter.Items))
		require.NotNil(t, iter.Metadata.SkipToken, "expected a next skip token for the next page")
		require.NotEmpty(t, *iter.Metadata.SkipToken)
	})

	// 2. Second page: limit=2, with skip token from page 1
	require.NotNil(t, iter.Metadata.SkipToken, "skip token for page 2 should not be nil")
	skipTokenPage2 := *iter.Metadata.SkipToken
	var iter2 *sdkregion.RegionIterator
	t.Run("second_page", func(t *testing.T) {
		iter2, err = rc.ListRegions(ctx, sdkregion.ListRegionsParams{Limit: &limit, SkipToken: &skipTokenPage2})
		require.NoError(t, err)
		require.NotNil(t, iter2)

		require.Len(t, iter2.Items, 2)
		require.ElementsMatch(t, regionNames[2:4], extractNames(iter2.Items))
		require.NotNil(t, iter2.Metadata.SkipToken, "expected a next skip token for the final page")
		require.NotEmpty(t, *iter2.Metadata.SkipToken)
	})

	// 3. Third page (final): limit=2, with skip token from page 2
	require.NotNil(t, iter2.Metadata.SkipToken, "skip token for page 3 should not be nil")
	skipTokenPage3 := *iter2.Metadata.SkipToken
	t.Run("third_and_final_page", func(t *testing.T) {
		iter, err := rc.ListRegions(ctx, sdkregion.ListRegionsParams{Limit: &limit, SkipToken: &skipTokenPage3})
		require.NoError(t, err)
		require.NotNil(t, iter)

		require.Len(t, iter.Items, 1)
		require.ElementsMatch(t, regionNames[4:5], extractNames(iter.Items))
		require.Nil(t, iter.Metadata.SkipToken, "did not expect a next skip token on the final page")
	})

	// 4. Limit larger than total items
	t.Run("limit_larger_than_total", func(t *testing.T) {
		largeLimit := 10
		iter, err := rc.ListRegions(ctx, sdkregion.ListRegionsParams{Limit: &largeLimit})
		require.NoError(t, err)
		require.NotNil(t, iter)

		require.Len(t, iter.Items, 5)
		require.ElementsMatch(t, regionNames, extractNames(iter.Items))
		require.Nil(t, iter.Metadata.SkipToken, "did not expect a next skip token when all items are returned")
	})
}

func extractNames(regs []schema.Region) []string {
	out := make([]string, len(regs))
	for i, r := range regs {
		out[i] = r.Metadata.Name
	}
	sort.Strings(out)
	return out
}

// --- Region CR construction helpers ---

type providerSpec struct {
	Name, Url, Version string
}

func newRegionCR(name string, labels map[string]string, az []string, providers []providerSpec, setVersionAndTimestamp bool) *regionsv1.Region {
	// Ensure required fields are populated to satisfy CRD validation when creating via API server.
	if len(az) == 0 {
		az = []string{"az-1"}
	}
	if len(providers) == 0 {
		providers = []providerSpec{{Name: "default", Url: "https://default", Version: "v1"}}
	}
	cr := &regionsv1.Region{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Region",
			APIVersion: regionsv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    labels,
			Namespace: "", // cluster-scoped

		},
		Spec: generatedv1.RegionSpec{
			AvailableZones: az,
			Providers:      make([]generatedv1.Provider, len(providers)),
		},
	}
	// should not be used with envtest(e2e)
	if setVersionAndTimestamp {
		cr.SetCreationTimestamp(metav1.Time{
			Time: time.Unix(1700000000, 0),
		})
		cr.SetResourceVersion("1")
	}
	for i, p := range providers {
		cr.Spec.Providers[i] = generatedv1.Provider{
			Name:    p.Name,
			Url:     p.Url,
			Version: p.Version,
		}
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
