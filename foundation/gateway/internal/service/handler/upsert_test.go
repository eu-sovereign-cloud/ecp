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
	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/service/handler"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
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

// Mock Updater
type MockUpdater[T any] struct {
	mock.Mock
}

func (m *MockUpdater[T]) Do(ctx context.Context, resource T) (T, error) {
	args := m.Called(ctx, resource)
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

func TestHandleUpsert(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	apiToDomain := func(sdk TestIn, locator handler.RegionalResourceLocator) TestDomain {
		return TestDomain{ID: locator.GetName(), Data: sdk.Data}
	}

	domainToAPI := func(domain TestDomain) TestOut {
		return TestOut(domain)
	}

	t.Run("success_create", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}
		outObj := TestOut{ID: "test-resource", Data: "test-data"}

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		mockLocator.On("GetWorkspace").Return("test-workspace")
		mockCreator.On("Do", mock.Anything, domainObj).Return(domainObj, nil)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert[TestIn, TestDomain, TestOut](rr, req, logger, mockLocator, mockCreator, mockUpdater, apiToDomain, domainToAPI)

		assert.Equal(t, http.StatusOK, rr.Code)

		var respBody TestOut
		err := json.Unmarshal(rr.Body.Bytes(), &respBody)
		require.NoError(t, err)
		assert.Equal(t, outObj, respBody)

		mockCreator.AssertExpectations(t)
		mockUpdater.AssertNotCalled(t, "Do")
		mockLocator.AssertExpectations(t)
	})

	t.Run("invalid_json", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		mockLocator.On("GetWorkspace").Return("")

		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader([]byte("invalid-json")))
		rr := httptest.NewRecorder()

		handler.HandleUpsert[TestIn, TestDomain, TestOut](rr, req, logger, mockLocator, mockCreator, mockUpdater, apiToDomain, domainToAPI)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid JSON in request body")
		mockCreator.AssertNotCalled(t, "Do")
		mockUpdater.AssertNotCalled(t, "Do")
	})

	t.Run("update_succeeds_on_already_exists", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		inObj := TestIn{Data: "updated-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "updated-data"}
		updatedDomainObj := TestDomain{ID: "test-resource", Data: "updated-data-from-updater"}
		outObj := TestOut{ID: "test-resource", Data: "updated-data-from-updater"}

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		mockLocator.On("GetWorkspace").Return("")

		errAlreadyExists := model.ErrAlreadyExists
		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, errAlreadyExists)
		mockUpdater.On("Do", mock.Anything, domainObj).Return(updatedDomainObj, nil)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert[TestIn, TestDomain, TestOut](rr, req, logger, mockLocator, mockCreator, mockUpdater, apiToDomain, domainToAPI)

		assert.Equal(t, http.StatusOK, rr.Code)
		var respBody TestOut
		err := json.Unmarshal(rr.Body.Bytes(), &respBody)
		require.NoError(t, err)
		assert.Equal(t, outObj, respBody)

		mockCreator.AssertExpectations(t)
		mockUpdater.AssertExpectations(t)
	})

	t.Run("update_fails_on_already_exists", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		mockLocator.On("GetWorkspace").Return("")

		errAlreadyExists := model.ErrAlreadyExists
		errUpdateFailed := errors.New("update failed")
		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, errAlreadyExists)
		mockUpdater.On("Do", mock.Anything, domainObj).Return(nil, errUpdateFailed)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert[TestIn, TestDomain, TestOut](rr, req, logger, mockLocator, mockCreator, mockUpdater, apiToDomain, domainToAPI)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockCreator.AssertExpectations(t)
		mockUpdater.AssertExpectations(t)
	})

	t.Run("creator_fails_other_error", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		mockLocator.On("GetWorkspace").Return("")

		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, errors.New("internal error"))

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert[TestIn, TestDomain, TestOut](rr, req, logger, mockLocator, mockCreator, mockUpdater, apiToDomain, domainToAPI)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockCreator.AssertExpectations(t)
		mockUpdater.AssertNotCalled(t, "Do")
	})

	t.Run("bad_request_body", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		mockLocator.On("GetWorkspace").Return("")

		rr := httptest.NewRecorder()
		// Create a request with a small body and then wrap it with MaxBytesReader to force a read error
		origReq := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader([]byte("x")))
		origReq.Body = http.MaxBytesReader(rr, origReq.Body, 0) // limit 0 bytes to trigger error on Read

		handler.HandleUpsert[TestIn, TestDomain, TestOut](rr, origReq, logger, mockLocator, mockCreator, mockUpdater, apiToDomain, domainToAPI)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to read request body")
		mockCreator.AssertNotCalled(t, "Do")
	})

	t.Run("encode_response_fails", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])
		mockLocator := new(MockRegionalResourceLocator)

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockLocator.On("GetName").Return("test-resource")
		mockLocator.On("GetTenant").Return("test-tenant")
		mockLocator.On("GetWorkspace").Return("")

		mockCreator.On("Do", mock.Anything, domainObj).Return(domainObj, nil)

		// This will cause json.Marshal to fail
		failingDomainToAPI := func(domain TestDomain) interface{} {
			return func() {}
		}

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert[TestIn, TestDomain, interface{}](rr, req, logger, mockLocator, mockCreator, mockUpdater, apiToDomain, failingDomainToAPI)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockCreator.AssertExpectations(t)
	})
}
