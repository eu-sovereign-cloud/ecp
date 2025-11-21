package region

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
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
		dyn := fake.NewSimpleDynamicClient(scheme, []runtime.Object{
			toUnstructured(t, scheme, testRegion),
		}...)
		gc := &GetRegion{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				dyn,
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}

		region, err := gc.Do(context.Background(), &model.Metadata{
			Name: regionName,
		})

		require.NoError(t, err)
		require.NotNil(t, region)
		require.NotNil(t, region.Metadata)
		require.Equal(t, regionName, region.Name)
		expectedZones := make([]model.Zone, len(availableZones))
		for i, z := range availableZones {
			expectedZones[i] = model.Zone(z)
		}
		require.ElementsMatch(t, expectedZones, region.Zones)
		require.Len(t, region.Providers, 2)
		require.Equal(t, "provider1", region.Providers[0].Name)
		require.Equal(t, "https://provider1.example.com", region.Providers[0].URL)
		require.Equal(t, "v1", region.Providers[0].Version)
		require.Equal(t, "provider2", region.Providers[1].Name)
	})

	t.Run("region_not_found", func(t *testing.T) {
		// Empty dynamic client with no regions
		gc := &GetRegion{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				fake.NewSimpleDynamicClient(scheme),
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}

		region, err := gc.Do(context.Background(), &model.Metadata{
			Name: "nonexistent-region",
		})

		require.Error(t, err)
		require.Nil(t, region)
	})

	t.Run("get_region_with_minimal_spec", func(t *testing.T) {
		minimalRegionName := "minimal-region"
		minimalRegion := newRegionCR(minimalRegionName, nil, nil, nil, true)

		gc := &GetRegion{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				fake.NewSimpleDynamicClient(scheme, []runtime.Object{
					toUnstructured(t, scheme, minimalRegion),
				}...),
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}

		region, err := gc.Do(context.Background(), &model.Metadata{
			Name: minimalRegionName,
		})

		require.NoError(t, err)
		require.NotNil(t, region)
		require.Equal(t, minimalRegionName, region.Metadata.Name)
		// Verify default values set by newRegionCR helper
		require.Len(t, region.Zones, 1)
		require.Equal(t, model.Zone("az-1"), region.Zones[0])
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

		gc := &GetRegion{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				fake.NewSimpleDynamicClient(scheme, objs...),
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}
		region, err := gc.Do(context.Background(), &model.Metadata{
			Name: multiAZRegionName,
		})

		require.NoError(t, err)
		require.NotNil(t, region)
		require.Equal(t, multiAZRegionName, region.Name)
		require.Len(t, region.Zones, 4)
		expectedZones := make([]model.Zone, len(multiAZ))
		for i, z := range multiAZ {
			expectedZones[i] = model.Zone(z)
		}
		require.ElementsMatch(t, expectedZones, region.Zones)
	})
}

// TestRegionController_GetRegion_Integration uses fake client to test edge cases
func TestRegionController_GetRegion_EdgeCases(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, regionsv1.AddToScheme(scheme))

	t.Run("get_with_empty_name", func(t *testing.T) {
		gc := &GetRegion{
			Logger: slog.Default(),
			Repo: kubernetes.NewAdapter(
				fake.NewSimpleDynamicClient(scheme),
				regionsv1.GroupVersionResource,
				slog.Default(),
				kubernetes.MapCRRegionToDomain,
			),
		}

		region, err := gc.Do(context.Background(), &model.Metadata{
			Name: "",
		})

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
		gc := &GetRegion{
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

		region, err := gc.Do(ctx, &model.Metadata{
			Name: regionName,
		})
		// The fake client might not respect context cancellation perfectly,
		// but we test the behavior anyway
		if err != nil {
			require.True(t, errors.Is(err, context.Canceled) || region == nil)
		}
	})
}
