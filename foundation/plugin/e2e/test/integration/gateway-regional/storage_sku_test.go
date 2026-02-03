//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

func TestStorageSKU_API(t *testing.T) {
	t.Parallel()

	// Given: The regional gateway is running and the storage SKU has been deployed.
	require.NotNil(t, storageClient, "storage client should have been initialized")

	t.Run("should retrieve a list of all available storage skus", func(t *testing.T) {
		t.Parallel()

		//
		// When we call the ListSkus method for our test tenant
		resp, err := storageClient.ListSkus(context.Background(), testTenant, nil)

		//
		// Then the call should succeed and return the SKU created during deployment
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.NoError(t, err)

		var skuIterator storagev1.SkuIterator
		err = json.Unmarshal(body, &skuIterator)
		require.NoError(t, err)

		require.NotNil(t, skuIterator.Items, "items in response body should not be nil")
		require.GreaterOrEqual(t, len(skuIterator.Items), 1, "expected to find at least 1 SKU")

		// And: The list should contain the 'sku-1' defined in the deployment files.
		foundSKU := false
		for _, sku := range skuIterator.Items {
			if sku.Metadata != nil && sku.Metadata.Name == "sku-1" {
				foundSKU = true
				break
			}
		}
		require.True(t, foundSKU, "should have found 'sku-1'")
	})

	t.Run("should retrieve a single specified storage sku by name", func(t *testing.T) {
		t.Parallel()

		//
		// Given the name of the SKU we know exists
		const skuName = "sku-1"

		//
		// When we call the GetSku method with the specified name
		resp, err := storageClient.GetSku(context.Background(), testTenant, skuName)

		//
		// Then the call should succeed and return the correct SKU details
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.NoError(t, err)

		var sku schema.StorageSku
		err = json.Unmarshal(body, &sku)
		require.NoError(t, err)

		//
		// And the details of the retrieved SKU should match our expectations
		require.NotNil(t, sku.Metadata, "sku metadata should not be nil")
		require.Equal(t, skuName, sku.Metadata.Name, "retrieved sku name should match the requested name")

		require.NotNil(t, sku.Spec, "sku spec should not be nil")
		require.Equal(t, 5000, sku.Spec.Iops)
		require.Equal(t, "local-ssd", string(sku.Spec.Type))
	})
}
