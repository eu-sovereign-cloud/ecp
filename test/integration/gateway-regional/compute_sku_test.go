//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	computev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// computeSKU is the InstanceSKU provisioned by the test-data fixtures.
const computeSKU = "compute-sku-1"

func TestInstanceSKU_API(t *testing.T) {
	t.Run("should retrieve a list of all available instance skus", func(t *testing.T) {
		//
		// When we call the ListSkus method for our test tenant
		resp, err := computeClient.ListSkus(context.Background(), testTenant, nil)

		//
		// Then the call should succeed and return the SKU created during deployment
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.NoError(t, err)

		var skuIterator computev1.SkuIterator
		err = json.Unmarshal(body, &skuIterator)
		require.NoError(t, err)

		require.NotNil(t, skuIterator.Items, "items in response body should not be nil")
		require.GreaterOrEqual(t, len(skuIterator.Items), 1, "expected to find at least 1 SKU")

		// And: The list should contain the SKU defined in the deployment files.
		foundSKU := false
		for _, sku := range skuIterator.Items {
			if sku.Metadata != nil && sku.Metadata.Name == computeSKU {
				foundSKU = true
				break
			}
		}
		require.True(t, foundSKU, "should have found %q", computeSKU)
	})

	t.Run("should retrieve a single specified instance sku by name", func(t *testing.T) {
		//
		// When we call the GetSku method with the specified name
		resp, err := computeClient.GetSku(context.Background(), testTenant, computeSKU)

		//
		// Then the call should succeed and return the correct SKU details
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.NoError(t, err)

		var sku schema.InstanceSku
		err = json.Unmarshal(body, &sku)
		require.NoError(t, err)

		//
		// And the details of the retrieved SKU should match our expectations
		require.NotNil(t, sku.Metadata, "sku metadata should not be nil")
		require.Equal(t, computeSKU, sku.Metadata.Name, "retrieved sku name should match the requested name")

		require.NotNil(t, sku.Spec, "sku spec should not be nil")
		require.Equal(t, 4, sku.Spec.VCPU)
		require.Equal(t, 8, sku.Spec.Ram)
	})
}
