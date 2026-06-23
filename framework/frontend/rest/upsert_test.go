package rest_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
)

// ---------------------------------------------------------------------------
// Test input/output types
// ---------------------------------------------------------------------------

// TestIn is the SDK input type decoded from the JSON request body.
type TestIn struct {
	Data string `json:"data"`
}

// TestDomain is the domain model used in upsert tests.
type TestDomain struct {
	ID   string
	Data string
}

// TestOut is the SDK response type encoded into the JSON response body.
type TestOut struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

// badOut has a channel field that causes json.Encoder to fail.
type badOut struct {
	C chan int `json:"c"`
}

// ---------------------------------------------------------------------------
// testParams implements persistence.IdentifiableResource
// ---------------------------------------------------------------------------

type testParams struct {
	name      string
	tenant    string
	workspace string
	version   string
}

func (p *testParams) GetName() string      { return p.name }
func (p *testParams) GetTenant() string    { return p.tenant }
func (p *testParams) GetWorkspace() string { return p.workspace }
func (p *testParams) GetVersion() string   { return p.version }

// ---------------------------------------------------------------------------
// Testify mock implementations for Creator and Updater
// ---------------------------------------------------------------------------

// MockCreator is a generic testify mock for frest.Creator[T].
type MockCreator[T any] struct {
	mock.Mock
}

func (m *MockCreator[T]) Do(ctx context.Context, resource T) (T, error) {
	args := m.Called(ctx, resource)
	if v := args.Get(0); v != nil {
		return v.(T), args.Error(1)
	}
	var zero T
	return zero, args.Error(1)
}

// MockUpdater is a generic testify mock for frest.Updater[T].
type MockUpdater[T any] struct {
	mock.Mock
}

func (m *MockUpdater[T]) Do(ctx context.Context, resource T) (T, error) {
	args := m.Called(ctx, resource)
	if v := args.Get(0); v != nil {
		return v.(T), args.Error(1)
	}
	var zero T
	return zero, args.Error(1)
}

// ---------------------------------------------------------------------------
// Shared test helpers
// ---------------------------------------------------------------------------

// upsertParams is the default identity for create-path tests (no version).
var upsertParams = &testParams{
	name:      "test-resource",
	tenant:    "test-tenant",
	workspace: "test-workspace",
}

// upsertParamsWithVersion is the identity for update-path tests (has a version).
var upsertParamsWithVersion = &testParams{
	name:      "test-resource",
	tenant:    "test-tenant",
	workspace: "test-workspace",
	version:   "42",
}

func newUpsertRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPut, "/v1/resources/test-resource",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func domainToTestOut(d TestDomain) TestOut {
	return TestOut{ID: d.ID, Data: d.Data}
}

func apiToTestDomain(sdk TestIn, params persistence.IdentifiableResource) TestDomain {
	return TestDomain{ID: params.GetName(), Data: sdk.Data}
}

// errBodyReader is an io.ReadCloser that always returns an error on Read.
type errBodyReader struct{}

func (e *errBodyReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated read error")
}
func (e *errBodyReader) Close() error { return nil }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandleUpsert_SuccessCreate(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}
	expectedDomain := TestDomain{ID: "test-resource", Data: "hello"}
	creator.On("Do", mock.Anything, mock.Anything).Return(expectedDomain, nil)

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{"data":"hello"}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	creator.AssertExpectations(t)
	updater.AssertNotCalled(t, "Do")
}

func TestHandleUpsert_InvalidJSON(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{not json}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "\"status\":400")
	creator.AssertNotCalled(t, "Do")
	updater.AssertNotCalled(t, "Do")
}

func TestHandleUpsert_BadRequestBody(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}

	req := httptest.NewRequest(http.MethodPut, "/v1/resources/test-resource", &errBodyReader{})
	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, req, discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	creator.AssertNotCalled(t, "Do")
	updater.AssertNotCalled(t, "Do")
}

func TestHandleUpsert_UpdateSucceedsOnAlreadyExists(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}
	expectedDomain := TestDomain{ID: "test-resource", Data: "hello"}

	creator.On("Do", mock.Anything, mock.Anything).Return(TestDomain{}, kernel.ErrAlreadyExists)
	updater.On("Do", mock.Anything, mock.Anything).Return(expectedDomain, nil)

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{"data":"hello"}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams, // version == "" → create path → AlreadyExists → fallthrough update
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	creator.AssertExpectations(t)
	updater.AssertExpectations(t)
}

func TestHandleUpsert_UpdateFailsOnAlreadyExists(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}

	creator.On("Do", mock.Anything, mock.Anything).Return(TestDomain{}, kernel.ErrAlreadyExists)
	updater.On("Do", mock.Anything, mock.Anything).Return(TestDomain{}, errors.New("update failed"))

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{"data":"hello"}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	creator.AssertExpectations(t)
	updater.AssertExpectations(t)
}

func TestHandleUpsert_CreatorFailsOtherError(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}

	creator.On("Do", mock.Anything, mock.Anything).Return(TestDomain{}, errors.New("unexpected error"))

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{"data":"hello"}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	creator.AssertExpectations(t)
	updater.AssertNotCalled(t, "Do")
}

func TestHandleUpsert_EncodeResponseFails(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}
	expectedDomain := TestDomain{ID: "test-resource", Data: "hello"}
	creator.On("Do", mock.Anything, mock.Anything).Return(expectedDomain, nil)

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{"data":"hello"}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, badOut]{
			Params:  upsertParams,
			Creator: creator,
			Updater: updater,
			APIToDomain: func(sdk TestIn, params persistence.IdentifiableResource) TestDomain {
				return TestDomain{ID: params.GetName(), Data: sdk.Data}
			},
			// DomainToAPI returns an un-serializable type (channel field).
			DomainToAPI: func(_ TestDomain) badOut { return badOut{C: make(chan int)} },
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	creator.AssertExpectations(t)
	updater.AssertNotCalled(t, "Do")
}

func TestHandleUpsert_CreateFailsNotFound(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}

	creator.On("Do", mock.Anything, mock.Anything).Return(TestDomain{}, kernel.NewError(kernel.KindNotFound, nil))

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{"data":"x"}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "\"status\":404")
	assert.Contains(t, string(body), "\"type\":\"http://secapi.cloud/errors/resource-not-found\"")
	creator.AssertExpectations(t)
	updater.AssertNotCalled(t, "Do")
}

func TestHandleUpsert_CreateFailsValidation(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}

	creator.On("Do", mock.Anything, mock.Anything).Return(TestDomain{}, kernel.NewError(kernel.KindValidation, nil))

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{"data":"x"}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParams,
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "\"status\":422")
	assert.Contains(t, string(body), "\"type\":\"http://secapi.cloud/errors/validation-error\"")
	creator.AssertExpectations(t)
	updater.AssertNotCalled(t, "Do")
}

func TestHandleUpsert_UpdateFailsNotFound(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}

	updater.On("Do", mock.Anything, mock.Anything).Return(TestDomain{}, kernel.NewError(kernel.KindNotFound, nil))

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{"data":"x"}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParamsWithVersion, // non-empty version → direct update path
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "\"status\":404")
	assert.Contains(t, string(body), "\"type\":\"http://secapi.cloud/errors/resource-not-found\"")
	creator.AssertNotCalled(t, "Do")
	updater.AssertExpectations(t)
}

func TestHandleUpsert_UpdateFailsConflict(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}

	updater.On("Do", mock.Anything, mock.Anything).Return(TestDomain{}, kernel.NewError(kernel.KindConflict, nil))

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{"data":"x"}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParamsWithVersion,
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "\"status\":409")
	assert.Contains(t, string(body), "\"type\":\"http://secapi.cloud/errors/resource-conflict\"")
	creator.AssertNotCalled(t, "Do")
	updater.AssertExpectations(t)
}

func TestHandleUpsert_UpdateFailsValidation(t *testing.T) {
	creator := &MockCreator[TestDomain]{}
	updater := &MockUpdater[TestDomain]{}

	updater.On("Do", mock.Anything, mock.Anything).Return(TestDomain{}, kernel.NewError(kernel.KindValidation, nil))

	recorder := httptest.NewRecorder()
	frest.HandleUpsert(recorder, newUpsertRequest(`{"data":"x"}`), discardLogger(),
		frest.UpsertOptions[TestIn, TestDomain, TestOut]{
			Params:      upsertParamsWithVersion,
			Creator:     creator,
			Updater:     updater,
			APIToDomain: apiToTestDomain,
			DomainToAPI: domainToTestOut,
		},
	)

	resp := recorder.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "\"status\":422")
	assert.Contains(t, string(body), "\"type\":\"http://secapi.cloud/errors/validation-error\"")
	creator.AssertNotCalled(t, "Do")
	updater.AssertExpectations(t)
}
