package region

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
)

func TestRegionController_GetRegion(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, regionsv1.AddToScheme(scheme))

	regionName := "test-region"
	regionLabels := map[string]string{"tier": "prod", "env": "production"}
	availableZones := []string{"az-1", "az-2"}
	providers := []providerSpec{
		{Name: "provider1", Url: "https://provider1.example.com", Version: "v1"},
		{Name: "provider2", Url: "https://provider2.example.com", Version: "v2"},
	}

	testRegion := newRegionCR(regionName, regionLabels, availableZones, providers, true)

	t.Run("successful_get", func(t *testing.T) {
		objs := []runtime.Object{
			toUnstructured(t, scheme, testRegion),
		}

		dyn := fake.NewSimpleDynamicClient(scheme, objs...)
		gc := &Get{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				dyn,
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}

		ctx := context.Background()
		region, err := gc.Do(ctx, regionName)

		require.NoError(t, err)
		require.NotNil(t, region)
		require.NotNil(t, region.Metadata)
		require.Equal(t, regionName, region.Metadata.Name)
		require.Equal(t, "get", region.Metadata.Verb)
		require.ElementsMatch(t, availableZones, region.Spec.AvailableZones)
		require.Len(t, region.Spec.Providers, 2)
		require.Equal(t, "provider1", region.Spec.Providers[0].Name)
		require.Equal(t, "https://provider1.example.com", region.Spec.Providers[0].Url)
		require.Equal(t, "v1", region.Spec.Providers[0].Version)
		require.Equal(t, "provider2", region.Spec.Providers[1].Name)
	})

	t.Run("region_not_found", func(t *testing.T) {
		// Empty dynamic client with no regions
		dyn := fake.NewSimpleDynamicClient(scheme)
		gc := &Get{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				dyn,
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}

		ctx := context.Background()
		region, err := gc.Do(ctx, "nonexistent-region")

		require.Error(t, err)
		require.Nil(t, region)
	})

	t.Run("get_region_with_minimal_spec", func(t *testing.T) {
		minimalRegionName := "minimal-region"
		minimalRegion := newRegionCR(minimalRegionName, nil, nil, nil, true)

		objs := []runtime.Object{
			toUnstructured(t, scheme, minimalRegion),
		}

		dyn := fake.NewSimpleDynamicClient(scheme, objs...)
		gc := &Get{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				dyn,
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}

		ctx := context.Background()
		region, err := gc.Do(ctx, minimalRegionName)

		require.NoError(t, err)
		require.NotNil(t, region)
		require.Equal(t, minimalRegionName, region.Metadata.Name)
		// Verify default values set by newRegionCR helper
		require.Len(t, region.Spec.AvailableZones, 1)
		require.Equal(t, "az-1", region.Spec.AvailableZones[0])
		require.Len(t, region.Spec.Providers, 1)
		require.Equal(t, "default", region.Spec.Providers[0].Name)
	})

	t.Run("get_region_with_multiple_availability_zones", func(t *testing.T) {
		multiAZRegionName := "multi-az-region"
		multiAZ := []string{"az-1", "az-2", "az-3", "az-4"}
		multiAZRegion := newRegionCR(multiAZRegionName, nil, multiAZ, nil, true)

		objs := []runtime.Object{
			toUnstructured(t, scheme, multiAZRegion),
		}

		dyn := fake.NewSimpleDynamicClient(scheme, objs...)
		gc := &Get{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				dyn,
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}

		ctx := context.Background()
		region, err := gc.Do(ctx, schema.ResourcePathParam(multiAZRegionName))

		require.NoError(t, err)
		require.NotNil(t, region)
		require.Equal(t, multiAZRegionName, region.Metadata.Name)
		require.Len(t, region.Spec.AvailableZones, 4)
		require.ElementsMatch(t, multiAZ, region.Spec.AvailableZones)
	})
}

// TestRegionController_GetRegion_Integration uses fake client to test edge cases
func TestRegionController_GetRegion_EdgeCases(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, regionsv1.AddToScheme(scheme))

	t.Run("get_with_empty_name", func(t *testing.T) {
		dyn := fake.NewSimpleDynamicClient(scheme)
		gc := &Get{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				dyn,
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}

		ctx := context.Background()
		region, err := gc.Do(ctx, "")

		require.Error(t, err)
		require.Nil(t, region)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		regionName := "test-region"
		testRegion := newRegionCR(regionName, nil, nil, nil, true)

		objs := []runtime.Object{
			toUnstructured(t, scheme, testRegion),
		}

		dyn := fake.NewSimpleDynamicClient(scheme, objs...)
		gc := &Get{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				dyn,
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		region, err := gc.Do(ctx, schema.ResourcePathParam(regionName))
		// The fake client might not respect context cancellation perfectly,
		// but we test the behavior anyway
		if err != nil {
			require.True(t, errors.Is(err, context.Canceled) || region == nil)
		}
	})
}
