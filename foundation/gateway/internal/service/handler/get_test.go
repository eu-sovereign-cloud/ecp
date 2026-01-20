package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// testResource implements port.IdentifiableResource
type testResource struct {
	name      string
	tenant    string
	workspace string
}

func (r *testResource) GetName() string      { return r.name }
func (r *testResource) GetTenant() string    { return r.tenant }
func (r *testResource) GetWorkspace() string { return r.workspace }

// mockGetter is a generic mock implementing Getter[D]
// It returns the preset object or error.
// When err is set, obj is ignored.
type mockGetter[D any] struct {
	obj D
	err error
}

func (m *mockGetter[D]) Do(ctx context.Context, resource port.IdentifiableResource) (D, error) {
	return m.obj, m.err
}

// domain model and output DTO for tests
type domainModel struct {
	Value string
}

type outputDTO struct {
	Value string `json:"value"`
}

// badDTO includes an unexported channel field causing json encoding to fail intentionally.
type badDTO struct {
	Bad chan int `json:"bad"`
}

func TestHandleGet_Success(t *testing.T) {
	res := &testResource{name: "demo", tenant: "tenant1", workspace: "workspace1"}
	getter := &mockGetter[domainModel]{obj: domainModel{Value: "abc"}}
	//nolint:staticcheck // S1016 suppression: mapping clarifies domain->DTO transformation.
	mapper := func(d domainModel) outputDTO { return outputDTO{Value: d.Value} }

	req := httptest.NewRequest(http.MethodGet, "/v1/resources/demo", nil)
	recorder := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	HandleGet(recorder, req, logger, res, getter, mapper)

	resp := recorder.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		// Fail early if not expected status
		l := string(b)
		// Provide body for debugging
		t.Fatalf("expected status 200, got %d body=%s", resp.StatusCode, l)
	}
	if ct := resp.Header.Get("Content-Type"); ct == "" {
		t.Errorf("expected Content-Type to be set, got empty")
	}
	body, _ := io.ReadAll(resp.Body)
	// Body should be JSON for outputDTO
	if string(body) != "{\"value\":\"abc\"}\n" { // json.Encoder adds newline
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestHandleGet_NotFound(t *testing.T) {
	res := &testResource{name: "missing", tenant: "tenant1", workspace: "workspace1"}
	// simulate not found error using k8s errors so errors.IsNotFound matches
	nfErr := k8serrors.NewNotFound(schema.GroupResource{Group: "test.io", Resource: "things"}, res.GetName())
	getter := &mockGetter[domainModel]{err: nfErr}
	//nolint:staticcheck // S1016 suppression: mapping clarifies domain->DTO transformation.
	mapper := func(d domainModel) outputDTO { return outputDTO{Value: d.Value} }

	req := httptest.NewRequest(http.MethodGet, "/v1/resources/missing", nil)
	recorder := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	HandleGet(recorder, req, logger, res, getter, mapper)

	resp := recorder.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 404, got %d body=%s", resp.StatusCode, string(b))
	}

	// Now expects structured JSON error according to RFC 7807
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	// Check that response is JSON and contains the expected fields
	if !contains(bodyStr, "\"status\":404") {
		t.Errorf("expected status field in JSON response, got: %s", bodyStr)
	}
	if !contains(bodyStr, "\"type\":\"http://secapi.eu/errors/resource-not-found\"") {
		t.Errorf("expected type field in JSON response, got: %s", bodyStr)
	}
	if !contains(bodyStr, res.GetName()) {
		t.Errorf("expected resource name in error detail, got: %s", bodyStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestHandleGet_InternalError(t *testing.T) {
	res := &testResource{name: "demo", tenant: "tenant1", workspace: "workspace1"}
	getter := &mockGetter[domainModel]{err: errors.New("boom")}
	//nolint:staticcheck // S1016 suppression: mapping clarifies domain->DTO transformation.
	mapper := func(d domainModel) outputDTO { return outputDTO{Value: d.Value} }

	req := httptest.NewRequest(http.MethodGet, "/v1/resources/demo", nil)
	recorder := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	HandleGet(recorder, req, logger, res, getter, mapper)

	resp := recorder.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 500, got %d body=%s", resp.StatusCode, string(b))
	}

	// Now expects structured JSON error according to RFC 7807
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !contains(bodyStr, "\"status\":500") {
		t.Errorf("expected status field in JSON response, got: %s", bodyStr)
	}
	if !contains(bodyStr, "\"type\":\"http://secapi.eu/errors/internal-server-error\"") {
		t.Errorf("expected type field in JSON response, got: %s", bodyStr)
	}
}

func TestHandleGet_EncodingFailure(t *testing.T) {
	res := &testResource{name: "demo", tenant: "tenant1", workspace: "workspace1"}
	getter := &mockGetter[domainModel]{obj: domainModel{Value: "abc"}}
	// mapper returns a struct with channel field which json cannot encode
	mapper := func(d domainModel) badDTO { return badDTO{Bad: make(chan int)} }

	req := httptest.NewRequest(http.MethodGet, "/v1/resources/demo", nil)
	recorder := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	HandleGet(recorder, req, logger, res, getter, mapper)

	resp := recorder.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 500, got %d body=%s", resp.StatusCode, string(b))
	}
}
