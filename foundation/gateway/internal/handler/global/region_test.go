package globalhandler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1" //nolint:goimports
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pkgerrors "k8s.io/apimachinery/pkg/api/errors"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"
)

// mockRegionProvider mocks the RegionProvider interface.
type mockRegionProvider struct {
	listRegionsIterator *region.RegionIterator
	listRegionsErr      error
	getRegionResult     *schema.Region
	getRegionErr        error
}

func (m *mockRegionProvider) ListRegions(_ context.Context, _ region.ListRegionsParams) (*region.RegionIterator, error) {
	return m.listRegionsIterator, m.listRegionsErr
}

func (m *mockRegionProvider) GetRegion(_ context.Context, _ schema.ResourcePathParam) (*schema.Region, error) {
	return m.getRegionResult, m.getRegionErr
}

func TestRegionHandler_ListRegions(t *testing.T) {
	testRegions := []schema.Region{
		{
			Metadata: nil,
			Spec: schema.RegionSpec{
				AvailableZones: []string{"zone1", "zone2"},
				Providers: []schema.Provider{
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
			listRegionsIterator: &region.RegionIterator{
				Items:    testRegions,
				Metadata: schema.ResponseMetadata{},
			},
		}
		handler := NewRegion(slog.Default(), mockProvider)
		req := httptest.NewRequest(http.MethodGet, "/regions", nil)
		rr := httptest.NewRecorder()

		// Act
		handler.ListRegions(rr, req, region.ListRegionsParams{})

		// Assert
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		var body region.RegionIterator
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, testRegions, body.Items)
	})

	t.Run("provider returns error", func(t *testing.T) {
		// Arrange
		mockProvider := &mockRegionProvider{
			listRegionsErr: errors.New("provider failed"),
		}
		handler := NewRegion(slog.Default(), mockProvider)
		req := httptest.NewRequest(http.MethodGet, "/regions", nil)
		rr := httptest.NewRecorder()

		// Act
		handler.ListRegions(rr, req, region.ListRegionsParams{})

		// Assert
		require.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to list regions: provider failed")
	})
}

func TestRegionHandler_GetRegion(t *testing.T) {
	testRegion := &schema.Region{
		Metadata: nil,
		Spec: schema.RegionSpec{
			AvailableZones: []string{"zone1", "zone2"},
			Providers: []schema.Provider{
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
		handler := NewRegion(slog.Default(), mockProvider)
		req := httptest.NewRequest(http.MethodGet, "/regions/eu-west-1", nil)
		rr := httptest.NewRecorder()

		// Act
		handler.GetRegion(rr, req, "eu-west-1")

		// Assert
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		var body *schema.Region
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, testRegion, body)
	})

	t.Run("internal server error, crd not installed", func(t *testing.T) {
		// Arrange
		mockProvider := &mockRegionProvider{
			getRegionErr: errors.New("failed to get region: CRD not installed"),
		}
		handler := NewRegion(slog.Default(), mockProvider)
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
			getRegionErr: pkgerrors.NewNotFound(regionsv1.GroupResource, "not found"),
		}
		handler := NewRegion(slog.Default(), mockProvider)
		req := httptest.NewRequest(http.MethodGet, "/regions/non-existent", nil)
		rr := httptest.NewRecorder()

		// Act
		handler.GetRegion(rr, req, "non-existent")

		// Assert
		require.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "not found")
	})
}
