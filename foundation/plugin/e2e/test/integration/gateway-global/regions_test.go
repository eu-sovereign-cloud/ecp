//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

func TestRegionAPI(t *testing.T) {
	//t.Parallel()

	//
	// Given the global gateway is running and has registered regions from the kustomize deployment.
	require.NotNil(t, regionClient, "region client should have been initialized")

	t.Run("should retrieve a list of all available regions", func(t *testing.T) {
		//t.Parallel()

		//
		// When we call the ListRegions method on the SDK client.
		resp, err := regionClient.ListRegions(context.Background(), nil)

		//
		// Then the call should succeed and return the expected regions.
		require.NoError(t, err, "listing regions should not produce an error")
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.NoError(t, err)

		var regionIterator regionv1.RegionIterator
		err = json.Unmarshal(body, &regionIterator)
		require.NoError(t, err, "failed to unmarshal list regions response")

		require.NotNil(t, regionIterator.Items, "items in response body should not be nil")

		//
		// And the list should contain the two regions defined in the deployment files.
		require.Len(t, regionIterator.Items, 2, "expected to find 2 regions")

		foundRegionOne := false
		foundRegionTwo := false
		for _, region := range regionIterator.Items {
			if region.Metadata != nil && region.Metadata.Name == "region-one" {
				foundRegionOne = true
			}
			if region.Metadata != nil && region.Metadata.Name == "region-two" {
				foundRegionTwo = true
			}
		}
		require.True(t, foundRegionOne, "should have found 'region-one'")
		require.True(t, foundRegionTwo, "should have found 'region-two'")
	})

	t.Run("should retrieve a single specified region by name", func(t *testing.T) {
		//t.Parallel()

		//
		// Given a specific region name we know exists.
		const regionName = "region-one"

		//
		// When we call the GetRegion method with the specified name.
		resp, err := regionClient.GetRegion(context.Background(), regionName)

		//
		// Then the call should succeed and return the correct region details.
		require.NoError(t, err, "getting a single region should not produce an error")
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.NoError(t, err)

		var region schema.Region
		err = json.Unmarshal(body, &region)
		require.NoError(t, err, "failed to unmarshal get region response")

		//
		// And the details of the retrieved region should match our expectations.
		require.NotNil(t, region.Metadata, "region metadata should not be nil")
		require.Equal(t, regionName, region.Metadata.Name, "retrieved region name should match the requested name")

		require.NotNil(t, region.Spec.Providers, "region should have providers")
		require.Len(t, region.Spec.Providers, 1, "expected 1 provider for region-one")
		require.Equal(t, "seca.compute", region.Spec.Providers[0].Name, "provider name should match the definition")

		require.NotNil(t, region.Spec.AvailableZones, "region should have available zones")
		require.Len(t, region.Spec.AvailableZones, 2, "expected 2 available zones for region-one")
		require.ElementsMatch(t, []string{"region-one-a", "region-one-b"}, region.Spec.AvailableZones, "available zones should match the definition")
	})
}
