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

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

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

func TestHandleUpsert(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	apiToDomain := func(sdk TestIn, params port.IdentifiableResource) TestDomain {
		return TestDomain{ID: params.GetName(), Data: sdk.Data}
	}

	domainToAPI := func(domain TestDomain) TestOut {
		return TestOut(domain)
	}

	t.Run("success_create", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])
		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}
		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}
		outObj := TestOut{ID: "test-resource", Data: "test-data"}

		mockCreator.On("Do", mock.Anything, domainObj).Return(domainObj, nil)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

		assert.Equal(t, http.StatusOK, rr.Code)

		var respBody TestOut
		err := json.Unmarshal(rr.Body.Bytes(), &respBody)
		require.NoError(t, err)
		assert.Equal(t, outObj, respBody)

		mockCreator.AssertExpectations(t)
		mockUpdater.AssertNotCalled(t, "Do")
	})

	t.Run("invalid_json", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}

		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader([]byte("invalid-json")))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		// Now returns structured JSON error
		assert.Contains(t, rr.Body.String(), "\"status\":400")
		assert.Contains(t, rr.Body.String(), "Invalid JSON")
		mockCreator.AssertNotCalled(t, "Do")
		mockUpdater.AssertNotCalled(t, "Do")
	})

	t.Run("update_succeeds_on_already_exists", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}

		inObj := TestIn{Data: "updated-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "updated-data"}
		updatedDomainObj := TestDomain{ID: "test-resource", Data: "updated-data-from-updater"}
		outObj := TestOut{ID: "test-resource", Data: "updated-data-from-updater"}

		errAlreadyExists := model.ErrAlreadyExists
		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, errAlreadyExists)
		mockUpdater.On("Do", mock.Anything, domainObj).Return(updatedDomainObj, nil)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

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

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		errAlreadyExists := model.ErrAlreadyExists
		errUpdateFailed := errors.New("update failed")
		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, errAlreadyExists)
		mockUpdater.On("Do", mock.Anything, domainObj).Return(nil, errUpdateFailed)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockCreator.AssertExpectations(t)
		mockUpdater.AssertExpectations(t)
	})

	t.Run("creator_fails_other_error", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, errors.New("internal error"))

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockCreator.AssertExpectations(t)
		mockUpdater.AssertNotCalled(t, "Do")
	})

	t.Run("bad_request_body", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}

		rr := httptest.NewRecorder()
		// Create a request with a small body and then wrap it with MaxBytesReader to force a read error
		origReq := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader([]byte("x")))
		origReq.Body = http.MaxBytesReader(rr, origReq.Body, 0) // limit 0 bytes to trigger error on Read

		handler.HandleUpsert(rr, origReq, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		// Now returns structured JSON error
		assert.Contains(t, rr.Body.String(), "\"status\":400")
		assert.Contains(t, rr.Body.String(), "Failed to read")
		mockCreator.AssertNotCalled(t, "Do")
	})

	t.Run("encode_response_fails", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockCreator.On("Do", mock.Anything, domainObj).Return(domainObj, nil)

		// This will cause json.Marshal to fail
		failingDomainToAPI := func(domain TestDomain) interface{} {
			return func() {}
		}

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, any]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: failingDomainToAPI,
		})
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockCreator.AssertExpectations(t)
	})

	t.Run("create_fails_not_found", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, model.ErrNotFound)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

		assert.Equal(t, http.StatusNotFound, rr.Code)
		mockCreator.AssertExpectations(t)
		mockUpdater.AssertNotCalled(t, "Do")
	})

	t.Run("create_fails_validation", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}
		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, model.ErrValidation)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

		assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
		mockCreator.AssertExpectations(t)
		mockUpdater.AssertNotCalled(t, "Do")
	})

	t.Run("update_fails_not_found", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, model.ErrAlreadyExists)
		mockUpdater.On("Do", mock.Anything, domainObj).Return(nil, model.ErrNotFound)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

		assert.Equal(t, http.StatusNotFound, rr.Code)
		mockCreator.AssertExpectations(t)
		mockUpdater.AssertExpectations(t)
	})

	t.Run("update_fails_conflict", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, model.ErrAlreadyExists)
		mockUpdater.On("Do", mock.Anything, domainObj).Return(nil, model.ErrConflict)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

		// ErrConflict now correctly maps to 409 Conflict per RFC 7807 spec
		assert.Equal(t, http.StatusConflict, rr.Code)
		mockCreator.AssertExpectations(t)
		mockUpdater.AssertExpectations(t)
	})

	t.Run("update_fails_validation", func(t *testing.T) {
		mockCreator := new(MockCreator[TestDomain])
		mockUpdater := new(MockUpdater[TestDomain])

		upsertParams := &regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "test-resource",
			},
			Scope: scope.Scope{
				Tenant:    "test-tenant",
				Workspace: "test-workspace",
			},
		}

		inObj := TestIn{Data: "test-data"}
		domainObj := TestDomain{ID: "test-resource", Data: "test-data"}

		mockCreator.On("Do", mock.Anything, domainObj).Return(nil, model.ErrAlreadyExists)
		mockUpdater.On("Do", mock.Anything, domainObj).Return(nil, model.ErrValidation)

		body, _ := json.Marshal(inObj)
		req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		handler.HandleUpsert(rr, req, logger, handler.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     mockCreator,
			Updater:     mockUpdater,
			SDKToDomain: apiToDomain,
			DomainToSDK: domainToAPI,
		})

		assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
		mockCreator.AssertExpectations(t)
		mockUpdater.AssertExpectations(t)
	})
}
