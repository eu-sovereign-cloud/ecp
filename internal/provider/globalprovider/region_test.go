package globalprovider

import (
	"context"
	"log/slog"
	"testing"
	"time"

	regionsapi "github.com/eu-sovereign-cloud/ecp/apis/regions"
	regionsv1 "github.com/eu-sovereign-cloud/ecp/apis/regions/crds/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	sdkregion "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"

	generatedv1 "github.com/eu-sovereign-cloud/ecp/apis/generated/types/region/v1"
	"github.com/eu-sovereign-cloud/ecp/internal/kubeclient"
)

func TestRegionController_ListRegions(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, regionsapi.AddToScheme(scheme))
	regionAName := "region-a"
	regionBName := "region-b"
	regionCName := "region-c"

	r1 := newRegionCR(regionAName, map[string]string{"tier": "prod", "env": "prod"}, []string{"az-a1"}, []providerSpec{{Name: "p1", Url: "https://p1", Version: "v1"}})
	r2 := newRegionCR(regionBName, map[string]string{"tier": "dev", "env": "staging"}, []string{"az-b1", "az-b2"}, []providerSpec{{Name: "p2", Url: "https://p2", Version: "v2"}})
	r3 := newRegionCR(regionCName, map[string]string{"tier": "prod", "env": "staging", "region": "3"}, []string{"az-c1"}, []providerSpec{{Name: "p3", Url: "https://p3", Version: "v3"}})

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
			regions, err := iter.All(ctx)
			require.NoError(t, err)
			require.ElementsMatch(t, tt.wantNames, extractNames(regions))
		})
	}
}

func extractNames(regs []*sdkregion.Region) []string {
	out := make([]string, len(regs))
	for i, r := range regs {
		out[i] = r.Metadata.Name
	}
	return out
}

// --- Region CR construction helpers ---

type providerSpec struct {
	Name, Url, Version string
}

func newRegionCR(name string, labels map[string]string, az []string, providers []providerSpec) *regionsv1.Region {
	cr := &regionsv1.Region{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Regions",
			APIVersion: regionsapi.Group + "/" + regionsapi.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    labels,
			Namespace: "", // cluster-scoped
			CreationTimestamp: metav1.Time{
				Time: time.Unix(1700000000, 0),
			},
			ResourceVersion: "1",
		},
		Spec: generatedv1.RegionSpec{
			AvailableZones: az,
			Providers:      make([]generatedv1.Provider, len(providers)),
		},
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
	gvks, _, err := scheme.ObjectKinds(obj)
	require.NoError(t, err)
	gvk := gvks[0]

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err)

	u := &unstructured.Unstructured{Object: m}
	u.SetGroupVersionKind(gvk)
	return u
}
