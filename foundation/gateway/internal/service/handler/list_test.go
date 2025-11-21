package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// mockLister is a generic mock implementing Lister[D]
type mockLister[D any] struct {
	objs          []D
	nextSkipToken *string
	err           error
}

func (m *mockLister[D]) Do(ctx context.Context, params model.ListParams) ([]D, *string, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.objs, m.nextSkipToken, nil
}

// listDomainModel and listOutputDTO for tests
type listDomainModel struct {
	Value string
}

type listOutputDTO struct {
	Items         []listDomainModel `json:"items"`
	NextSkipToken *string           `json:"nextSkipToken,omitempty"`
}

func TestHandleList_Success(t *testing.T) {
	lister := &mockLister[listDomainModel]{
		objs: []listDomainModel{{Value: "a"}, {Value: "b"}},
	}
	mapper := func(d []listDomainModel, next *string) listOutputDTO {
		return listOutputDTO{Items: d, NextSkipToken: next}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	recorder := httptest.NewRecorder()

	HandleList(recorder, req, slog.New(slog.NewTextHandler(io.Discard, nil)), model.ListParams{}, lister, mapper)

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	expectedBody := `{"items":[{"Value":"a"},{"Value":"b"}]}` + "\n"
	if string(body) != expectedBody {
		t.Errorf("unexpected body: got %q, want %q", string(body), expectedBody)
	}
}

func TestHandleList_InternalError(t *testing.T) {
	lister := &mockLister[listDomainModel]{err: errors.New("db connection failed")}
	mapper := func(d []listDomainModel, next *string) listOutputDTO {
		return listOutputDTO{Items: d, NextSkipToken: next}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	recorder := httptest.NewRecorder()
	HandleList(recorder, req, slog.New(slog.NewTextHandler(io.Discard, nil)), model.ListParams{}, lister, mapper)

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}
}

type badListDTO struct {
	Bad chan int `json:"bad"`
}

func TestHandleList_EncodingFailure(t *testing.T) {
	lister := &mockLister[listDomainModel]{
		objs: []listDomainModel{{Value: "a"}},
	}
	mapper := func(d []listDomainModel, next *string) badListDTO {
		return badListDTO{Bad: make(chan int)}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	recorder := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	HandleList(recorder, req, logger, model.ListParams{}, lister, mapper)

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}
}
