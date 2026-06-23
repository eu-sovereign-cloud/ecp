package rest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

// mockLister is a generic mock implementing Lister[D].
type mockLister[D any] struct {
	items     []D
	nextToken *string
	err       error
}

func (m *mockLister[D]) Do(_ context.Context, _ resource.ListParams) ([]D, *string, error) {
	return m.items, m.nextToken, m.err
}

// listDTO is the response envelope for list tests.
type listDTO struct {
	Items []outputDTO `json:"items"`
}

// domainToListDTO converts a slice of domainModel into listDTO (satisfies DomainToAPIList[domainModel,listDTO]).
func domainToListDTO(items []domainModel, _ *string) listDTO {
	dtos := make([]outputDTO, len(items))
	for i, d := range items {
		dtos[i] = outputDTO(d)
	}
	return listDTO{Items: dtos}
}

func TestHandleList_Success(t *testing.T) {
	lister := &mockLister[domainModel]{
		items: []domainModel{{Value: "a"}, {Value: "b"}},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	recorder := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	HandleList(recorder, req, logger, resource.ListParams{}, lister, domainToListDTO)

	resp := recorder.Result()
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d body=%s", resp.StatusCode, string(b))
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "\"a\"") || !strings.Contains(bodyStr, "\"b\"") {
		t.Errorf("expected both items in response, got: %s", bodyStr)
	}
}

func TestHandleList_InternalError(t *testing.T) {
	lister := &mockLister[domainModel]{
		err: errors.New("internal error"),
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	recorder := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	HandleList(recorder, req, logger, resource.ListParams{}, lister, domainToListDTO)

	resp := recorder.Result()
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusInternalServerError {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 500, got %d body=%s", resp.StatusCode, string(b))
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "\"status\":500") {
		t.Errorf("expected status field in JSON response, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "\"type\":\"http://secapi.cloud/errors/internal-server-error\"") {
		t.Errorf("expected type field in JSON response, got: %s", bodyStr)
	}
}

func TestHandleList_EncodingFailure(t *testing.T) {
	// badListDTO wraps a badDTO slice so it cannot be encoded.
	type badListDTO struct {
		Items []badDTO `json:"items"`
	}
	lister := &mockLister[domainModel]{
		items: []domainModel{{Value: "x"}},
	}
	// mapper returns a struct with a channel field which json cannot encode.
	mapper := func(_ []domainModel, _ *string) badListDTO {
		return badListDTO{Items: []badDTO{{Bad: make(chan int)}}}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	recorder := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	HandleList(recorder, req, logger, resource.ListParams{}, lister, mapper)

	resp := recorder.Result()
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusInternalServerError {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 500, got %d body=%s", resp.StatusCode, string(b))
	}
}
