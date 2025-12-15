package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/service/handler"
)

// Mock Creator
type MockCreator[T any] struct {
	mock.Mock
}

func (m *MockCreator[T]) Do(ctx context.Context, resource T) (T, error) {
	args := m.Called(ctx, resource)
	// Handle nil case for the first return value if an error is expected.
	if args.Get(0) == nil {
		var zero T
		return zero, args.Error(1)
	}
	return args.Get(0).(T), args.Error(1)
}

// Test types
type TestIn struct {
	Data string `json:"data"`
}

type TestDomain struct {
	ID   string
	Data string
}

type TestOut struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

// Mock RegionalResourceLocator
type MockRegionalResourceLocator struct {
	mock.Mock
}

func (m *MockRegionalResourceLocator) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockRegionalResourceLocator) GetTenant() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockRegionalResourceLocator) GetWorkspace() string {
	args := m.Called()
	return args.String(0)
}

func TestHandleCreateOrUpdate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	apiToDomain := func(api TestIn, tenant, name string) TestDomain {
		return TestDomain{ID: name, Data: api.Data}
	}

	domainToAPI := func(domain TestDomain) TestOut {
		return TestOut{ID: domain.ID, Data: domain.Data}
	}

	t.Run("success", func(t *testing.T) {
		// Arrange
		mockCreator := new(MockCreator[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}
		outObj := TestOut{ID: "test-resource", Data: "test-data"}

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		mockCreator.On("Do", mock.Anything, domainObj).Return(domainObj, nil)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		// Act
		handler.HandleCreateOrUpdate[TestIn, TestDomain, TestOut](rr, req, logger, mockLocator, mockCreator, apiToDomain, domainToAPI)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var respBody TestOut
		err := json.Unmarshal(rr.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, outObj, respBody)

		mockCreator.AssertExpectations(t)
		mockLocator.AssertExpectations(t)
	})

	t.Run("invalid_json", func(t *testing.T) {
		// Arrange
		mockCreator := new(MockCreator[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")

		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader([]byte("invalid-json")))
		rr := httptest.NewRecorder()

		// Act
		handler.HandleCreateOrUpdate[TestIn, TestDomain, TestOut](rr, req, logger, mockLocator, mockCreator, apiToDomain, domainToAPI)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid JSON in request body")
		mockCreator.AssertNotCalled(t, "Do")
	})

	t.Run("already_exists", func(t *testing.T) {
		// Arrange
		mockCreator := new(MockCreator[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		errAlreadyExists := k8serrors.NewAlreadyExists(schema.GroupResource{}, "test-resource")
		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, errAlreadyExists)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		// Act
		handler.HandleCreateOrUpdate[TestIn, TestDomain, TestOut](rr, req, logger, mockLocator, mockCreator, apiToDomain, domainToAPI)

		// Assert
		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Contains(t, rr.Body.String(), "resource test-resource already exists")
		mockCreator.AssertExpectations(t)
	})

	t.Run("creator_fails", func(t *testing.T) {
		// Arrange
		mockCreator := new(MockCreator[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, errors.New("internal error"))

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		// Act
		handler.HandleCreateOrUpdate[TestIn, TestDomain, TestOut](rr, req, logger, mockLocator, mockCreator, apiToDomain, domainToAPI)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockCreator.AssertExpectations(t)
	})

	t.Run("bad_request_body", func(t *testing.T) {
		// Arrange
		mockCreator := new(MockCreator[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")

		rr := httptest.NewRecorder()
		// Create a request with a small body and then wrap it with MaxBytesReader to force a read error
		origReq := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader([]byte("x")))
		origReq.Body = http.MaxBytesReader(rr, origReq.Body, 0) // limit 0 bytes to trigger error on Read

		// Act
		handler.HandleCreateOrUpdate[TestIn, TestDomain, TestOut](rr, origReq, logger, mockLocator, mockCreator, apiToDomain, domainToAPI)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to read request body")
		mockCreator.AssertNotCalled(t, "Do")
	})

	t.Run("encode_response_fails", func(t *testing.T) {
		// Arrange
		mockCreator := new(MockCreator[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		mockCreator.On("Do", mock.Anything, domainObj).Return(domainObj, nil)

		// This will cause json.Marshal to fail
		failingDomainToAPI := func(domain TestDomain) interface{} {
			return func() {}
		}

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		// Act
		handler.HandleCreateOrUpdate[TestIn, TestDomain, interface{}](rr, req, logger, mockLocator, mockCreator, apiToDomain, failingDomainToAPI)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockCreator.AssertExpectations(t)
	})
}
