package region

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

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

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

var cfg *rest.Config // package-level test kubeconfig

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
	rc := &ListRegion{
		Logger: slog.Default(),
		Repo: kubernetes.NewAdapter(
			dyn,
			regionsv1.GroupVersionResource,
			slog.Default(),
			kubernetes.MapCRRegionToDomain,
		),
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
			params := model.ListParams{}
			if tt.selector != nil {
				params.Selector = *tt.selector
			}
			regions, _, err := rc.Do(ctx, params)
			require.NoError(t, err)
			require.ElementsMatch(t, tt.wantNames, extractDomainNames(regions))
		})
	}
}

func TestMain(m *testing.M) {
	// Resolve CRD directory path relative to this test file's package directory.
	wd, _ := os.Getwd()
	crdDir := filepath.Clean(filepath.Join(wd, "../../../../../api/generated/crds/regions"))

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

// TestRegionController_ListRegions_Pagination - uses envtest, as fake kube does not support limit
func TestRegionController_ListRegions_Pagination(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, regionsv1.AddToScheme(scheme))

	dynClient, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)

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
	rc := &ListRegion{
		Logger: slog.Default(),
		Repo: kubernetes.NewAdapter(
			dynClient,
			regionsv1.GroupVersionResource,
			slog.Default(),
			kubernetes.MapCRRegionToDomain,
		),
	}
	// 1. First page: limit=2, no skip token
	var skipToken *string
	var regions []*model.RegionDomain
	var next *string
	// first page
	t.Run("first_page", func(t *testing.T) {
		params := model.ListParams{Limit: limit}
		regions, next, err = rc.Do(ctx, params)
		require.NoError(t, err)
		require.Len(t, regions, 2)
		require.ElementsMatch(t, regionNames[0:2], extractDomainNames(regions))
		require.NotNil(t, next)
		require.NotEmpty(t, *next)
		skipToken = next
	})
	// second page
	require.NotNil(t, skipToken)
	skipTokenPage2 := *skipToken
	var secondPage []*model.RegionDomain
	var next2 *string
	t.Run("second_page", func(t *testing.T) {
		params := model.ListParams{Limit: limit, SkipToken: skipTokenPage2}
		secondPage, next2, err = rc.Do(ctx, params)
		require.NoError(t, err)
		require.Len(t, secondPage, 2)
		require.ElementsMatch(t, regionNames[2:4], extractDomainNames(secondPage))
		require.NotNil(t, next2)
		require.NotEmpty(t, *next2)
	})
	// third page
	require.NotNil(t, next2)
	skipTokenPage3 := *next2
	t.Run("third_and_final_page", func(t *testing.T) {
		params := model.ListParams{Limit: limit, SkipToken: skipTokenPage3}
		lastPage, next3, err := rc.Do(ctx, params)
		require.NoError(t, err)
		require.Len(t, lastPage, 1)
		require.ElementsMatch(t, regionNames[4:5], extractDomainNames(lastPage))
		require.Nil(t, next3)
	})
	// large limit
	t.Run("limit_larger_than_total", func(t *testing.T) {
		largeLimit := 10
		allRegions, nextFinal, err := rc.Do(ctx, model.ListParams{Limit: largeLimit})
		require.NoError(t, err)
		require.Len(t, allRegions, 5)
		require.ElementsMatch(t, regionNames, extractDomainNames(allRegions))
		require.Nil(t, nextFinal)
	})
}

func extractDomainNames(regs []*model.RegionDomain) []string {
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

	// Convert []string to []generatedv1.Zone for the CR spec
	zones := make([]generatedv1.Zone, len(az))
	for i, z := range az {
		zones[i] = z
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
			AvailableZones: zones,
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
