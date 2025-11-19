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

// testResource implements port.NamespacedResource
type testResource struct {
	name      string
	namespace string
}

func (r *testResource) GetName() string        { return r.name }
func (r *testResource) GetNamespace() string   { return r.namespace }
func (r *testResource) SetName(n string)       { r.name = n }
func (r *testResource) SetNamespace(ns string) { r.namespace = ns }

// mockGetter is a generic mock implementing Getter[D]
// It returns the preset object or error.
// When err is set, obj is ignored.
type mockGetter[D any] struct {
	obj D
	err error
}

func (m *mockGetter[D]) Do(ctx context.Context, resource port.NamespacedResource) (D, error) {
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
	res := &testResource{name: "demo", namespace: "ns"}
	getter := &mockGetter[domainModel]{obj: domainModel{Value: "abc"}}
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
	res := &testResource{name: "missing", namespace: "ns"}
	// simulate not found error using k8s errors so errors.IsNotFound matches
	nfErr := k8serrors.NewNotFound(schema.GroupResource{Group: "test.io", Resource: "things"}, res.GetName())
	getter := &mockGetter[domainModel]{err: nfErr}
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
	body, _ := io.ReadAll(resp.Body)
	// http.Error appends a newline
	if expected := res.GetName() + " not found\n"; string(body) != expected {
		t.Errorf("expected body %q, got %q", expected, string(body))
	}
}

func TestHandleGet_InternalError(t *testing.T) {
	res := &testResource{name: "demo", namespace: "ns"}
	getter := &mockGetter[domainModel]{err: errors.New("boom")}
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
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Internal Server Error\n" {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestHandleGet_EncodingFailure(t *testing.T) {
	res := &testResource{name: "demo", namespace: "ns"}
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
