package regionalhandler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// mockStorageProvider mocks regionalprovider.StorageProvider.
type mockStorageProvider struct {
	iter         *sdkstorage.SkuIterator
	listSkusErr  error
	getSkuResult *sdkschema.StorageSku
	getSkuErr    error
}

func (m *mockStorageProvider) CreateOrUpdateImage(ctx context.Context, tenantID, imageID string, params sdkstorage.CreateOrUpdateImageParams, req sdkschema.Image) (*sdkschema.Image, bool, error) {
	// TODO implement me
	panic("implement me")
}

func (m *mockStorageProvider) GetImage(ctx context.Context, tenantID, imageID string) (*sdkschema.Image, error) {
	// TODO implement me
	panic("implement me")
}

func (m *mockStorageProvider) DeleteImage(ctx context.Context, tenantID, imageID string, params sdkstorage.DeleteImageParams) error {
	// TODO implement me
	panic("implement me")
}

func (m *mockStorageProvider) ListImages(ctx context.Context, tenantID string, params sdkstorage.ListImagesParams) (*secapi.Iterator[sdkschema.Image], error) {
	// TODO implement me
	panic("implement me")
}

func (m *mockStorageProvider) ListBlockStorages(ctx context.Context, tenantID, workspaceID string, params sdkstorage.ListBlockStoragesParams) (*secapi.Iterator[sdkschema.BlockStorage], error) {
	// TODO implement me
	panic("implement me")
}

func (m *mockStorageProvider) GetBlockStorage(ctx context.Context, tenantID, workspaceID, storageID string) (*sdkschema.BlockStorage, error) {
	// TODO implement me
	panic("implement me")
}

func (m *mockStorageProvider) CreateOrUpdateBlockStorage(ctx context.Context, tenantID, workspaceID, storageID string, params sdkstorage.CreateOrUpdateBlockStorageParams, req sdkschema.BlockStorage) (*sdkschema.BlockStorage, bool, error) {
	// TODO implement me
	panic("implement me")
}

func (m *mockStorageProvider) DeleteBlockStorage(ctx context.Context, tenantID, workspaceID, storageID string, params sdkstorage.DeleteBlockStorageParams) error {
	// TODO implement me
	panic("implement me")
}

func (m *mockStorageProvider) ListSKUs(_ context.Context, _ sdkschema.TenantPathParam, _ sdkstorage.ListSkusParams) (*sdkstorage.SkuIterator, error) {
	return m.iter, m.listSkusErr
}

func (m *mockStorageProvider) GetSKU(_ context.Context, _ sdkschema.TenantPathParam, _ sdkschema.ResourcePathParam) (*sdkschema.StorageSku, error) {
	return m.getSkuResult, m.getSkuErr
}

func TestStorageHandler_ListSkus(t *testing.T) {
	testSkus := []sdkschema.StorageSku{
		{
			Metadata: &sdkschema.SkuResourceMetadata{
				Name: "standard-1",
			},
			Spec: &sdkschema.StorageSkuSpec{
				Iops:          0,
				MinVolumeSize: 0,
				Type:          "",
			},
		},
	}

	t.Run("success", func(t *testing.T) {
		mp := &mockStorageProvider{
			iter: &sdkstorage.SkuIterator{Items: testSkus},
		}
		handler := NewStorage(slog.Default(), mp)
		req := httptest.NewRequest(http.MethodGet, "/tenants/t1/storage/skus", nil)
		rr := httptest.NewRecorder()

		handler.ListSkus(rr, req, sdkschema.TenantPathParam("t1"), sdkstorage.ListSkusParams{})

		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		var skuIterator *sdkstorage.SkuIterator
		err := json.Unmarshal(rr.Body.Bytes(), &skuIterator)
		require.NoError(t, err)
		assert.Equal(t, testSkus, skuIterator.Items)
	})

	t.Run("provider error", func(t *testing.T) {
		mp := &mockStorageProvider{
			listSkusErr: errors.New("provider failed"),
		}
		handler := NewStorage(slog.Default(), mp)
		req := httptest.NewRequest(http.MethodGet, "/tenants/t1/storage/skus", nil)
		rr := httptest.NewRecorder()

		handler.ListSkus(rr, req, "t1", sdkstorage.ListSkusParams{})

		require.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to list storage skus: provider failed")
	})
}

func TestStorageHandler_GetSku(t *testing.T) {
	testSku := &sdkschema.StorageSku{
		Metadata: &sdkschema.SkuResourceMetadata{
			Name: "premium-1",
		},
		Spec: &sdkschema.StorageSkuSpec{
			Iops:          0,
			MinVolumeSize: 0,
			Type:          "",
		},
	}

	t.Run("success", func(t *testing.T) {
		mp := &mockStorageProvider{
			getSkuResult: testSku,
		}
		handler := NewStorage(slog.Default(), mp)
		req := httptest.NewRequest(http.MethodGet, "/tenants/t1/storage/skus/premium-1", nil)
		rr := httptest.NewRecorder()

		handler.GetSku(rr, req, "t1", "premium-1")

		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		var body *sdkschema.StorageSku
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, testSku, body)
	})

	t.Run("not found", func(t *testing.T) {
		mp := &mockStorageProvider{
			getSkuErr: k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: "storageskus"}, "missing"),
		}
		handler := NewStorage(slog.Default(), mp)
		req := httptest.NewRequest(http.MethodGet, "/tenants/t1/storage/skus/missing", nil)
		rr := httptest.NewRecorder()

		handler.GetSku(rr, req, "t1", "missing")

		require.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "storage sku (missing) not found")
	})

	t.Run("internal error", func(t *testing.T) {
		mp := &mockStorageProvider{
			getSkuErr: errors.New("backend unavailable"),
		}
		handler := NewStorage(slog.Default(), mp)
		req := httptest.NewRequest(http.MethodGet, "/tenants/t1/storage/skus/premium-1", nil)
		rr := httptest.NewRecorder()

		handler.GetSku(rr, req, "t1", "premium-1")

		require.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), http.StatusText(http.StatusInternalServerError))
	})
}
