package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1" //nolint:goimports
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pkgerrors "k8s.io/apimachinery/pkg/api/errors"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/apis/regions/crds/v1"
)

// mockRegionProvider mocks the RegionProvider interface.
type mockRegionProvider struct {
	listRegionsIterator *secapi.Iterator[region.Region]
	listRegionsErr      error
	getRegionResult     *region.Region
	getRegionErr        error
}

func (m *mockRegionProvider) ListRegions(_ context.Context, _ region.ListRegionsParams) (*secapi.Iterator[region.Region], error) {
	return m.listRegionsIterator, m.listRegionsErr
}

func (m *mockRegionProvider) GetRegion(_ context.Context, _ region.ResourceName) (*region.Region, error) {
	return m.getRegionResult, m.getRegionErr
}

func TestRegionHandler_ListRegions(t *testing.T) {
	testRegions := []region.Region{
		{
			Metadata: nil,
			Spec: region.RegionSpec{
				AvailableZones: []string{"zone1", "zone2"},
				Providers: []region.Provider{
					{
						Name:    "seca.compute",
						Url:     "url_here",
						Version: "v1",
					},
				},
			},
		},
	}

	t.Run("success", func(t *testing.T) {
		// Arrange
		mockProvider := &mockRegionProvider{
			listRegionsIterator: secapi.NewIterator(func(ctx context.Context, skipToken *string) ([]region.Region, *string, error) {
				return testRegions, skipToken, nil
			}),
		}
		handler := NewRegionHandler(slog.Default(), mockProvider)
		req := httptest.NewRequest(http.MethodGet, "/regions", nil)
		rr := httptest.NewRecorder()

		// Act
		handler.ListRegions(rr, req, region.ListRegionsParams{})

		// Assert
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		var body []region.Region
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, testRegions, body)
	})

	t.Run("provider returns error", func(t *testing.T) {
		// Arrange
		mockProvider := &mockRegionProvider{
			listRegionsErr: errors.New("provider failed"),
		}
		handler := NewRegionHandler(slog.Default(), mockProvider)
		req := httptest.NewRequest(http.MethodGet, "/regions", nil)
		rr := httptest.NewRecorder()

		// Act
		handler.ListRegions(rr, req, region.ListRegionsParams{})

		// Assert
		require.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to list regions: provider failed")
	})

	t.Run("iterator returns error", func(t *testing.T) {
		// Arrange
		mockProvider := &mockRegionProvider{
			listRegionsIterator: secapi.NewIterator(func(ctx context.Context, skipToken *string) ([]region.Region, *string, error) {
				return testRegions, skipToken, errors.New("iterator failed")
			}),
		}
		handler := NewRegionHandler(slog.Default(), mockProvider)
		req := httptest.NewRequest(http.MethodGet, "/regions", nil)
		rr := httptest.NewRecorder()

		// Act
		handler.ListRegions(rr, req, region.ListRegionsParams{})

		// Assert
		require.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to retrieve all regions: iterator failed")
	})
}

func TestRegionHandler_GetRegion(t *testing.T) {
	testRegion := &region.Region{
		Metadata: nil,
		Spec: region.RegionSpec{
			AvailableZones: []string{"zone1", "zone2"},
			Providers: []region.Provider{
				{
					Name:    "seca.compute",
					Url:     "url_here",
					Version: "v1",
				},
			},
		},
	}

	t.Run("success", func(t *testing.T) {
		// Arrange
		mockProvider := &mockRegionProvider{
			getRegionResult: testRegion,
		}
		handler := NewRegionHandler(slog.Default(), mockProvider)
		req := httptest.NewRequest(http.MethodGet, "/regions/eu-west-1", nil)
		rr := httptest.NewRecorder()

		// Act
		handler.GetRegion(rr, req, "eu-west-1")

		// Assert
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		var body *region.Region
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, testRegion, body)
	})

	t.Run("internal server error, crd not installed", func(t *testing.T) {
		// Arrange
		mockProvider := &mockRegionProvider{
			getRegionErr: errors.New("failed to get region: CRD not installed"),
		}
		handler := NewRegionHandler(slog.Default(), mockProvider)
		req := httptest.NewRequest(http.MethodGet, "/regions/non-existent", nil)
		rr := httptest.NewRecorder()

		// Act
		handler.GetRegion(rr, req, "non-existent")

		// Assert
		require.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), http.StatusText(http.StatusInternalServerError))
	})
	t.Run("not found", func(t *testing.T) {
		// Arrange
		mockProvider := &mockRegionProvider{
			getRegionErr: pkgerrors.NewNotFound(regionsv1.RegionGR, "not found"),
		}
		handler := NewRegionHandler(slog.Default(), mockProvider)
		req := httptest.NewRequest(http.MethodGet, "/regions/non-existent", nil)
		rr := httptest.NewRecorder()

		// Act
		handler.GetRegion(rr, req, "non-existent")

		// Assert
		require.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "not found")
	})
}
