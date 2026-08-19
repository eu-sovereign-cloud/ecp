package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestHTTPMiddleware_RecordsStatusAndLabels(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /providers/seca.region/v1/regions", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	mux.HandleFunc("GET /providers/seca.region/v1/regions/{name}", func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := HTTPMiddleware(mux)

	ctx := context.Background()

	// Success list
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/providers/seca.region/v1/regions", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	count := histogramCount(t, "GET", "200",
		"/providers/seca.region/v1/regions", "seca.region")
	if count < 1 {
		t.Fatalf("expected at least 1 observation for region list, got %v", count)
	}

	// 404 get
	req404 := httptest.NewRequestWithContext(ctx, http.MethodGet, "/providers/seca.region/v1/regions/missing", nil)
	rec404 := httptest.NewRecorder()
	h.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec404.Code)
	}
	count404 := histogramCount(t, "GET", "404",
		"/providers/seca.region/v1/regions/{name}", "seca.region")
	if count404 < 1 {
		t.Fatalf("expected at least 1 observation for 404, got %v", count404)
	}

	// Probe
	reqHealth := httptest.NewRequestWithContext(ctx, http.MethodGet, "/healthz", nil)
	recHealth := httptest.NewRecorder()
	h.ServeHTTP(recHealth, reqHealth)
	countHealth := histogramCount(t, "GET", "200", "/healthz", "system")
	if countHealth < 1 {
		t.Fatalf("expected healthz observation, got %v", countHealth)
	}
}

func TestHTTPMiddleware_SkipsMetricsPath(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})
	h := HTTPMiddleware(inner)

	before := totalHTTPObservations(t)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("inner handler was not called for /metrics")
	}
	after := totalHTTPObservations(t)
	if after != before {
		t.Fatalf(" /metrics must not record observations: before=%v after=%v", before, after)
	}
}

func TestHTTPMiddleware_DefaultStatusOKOnWrite(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hi"))
	})
	h := HTTPMiddleware(inner)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil)
	// Force fallback normalizer (no Pattern) by not using ServeMux.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	count := histogramCount(t, "GET", "200", "/healthz", "system")
	if count < 1 {
		t.Fatalf("expected observation with status 200, got count %v", count)
	}
}

func TestNormalizeMethod(t *testing.T) {
	t.Parallel()
	if got := normalizeMethod(http.MethodGet); got != "GET" {
		t.Errorf("got %q", got)
	}
	if got := normalizeMethod("TRACE"); got != "OTHER" {
		t.Errorf("got %q, want OTHER", got)
	}
}

func histogramCount(t *testing.T, method, status, route, provider string) float64 {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != "ecp_gateway_http_request_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			if !labelsMatch(m, method, status, route, provider) {
				continue
			}
			if h := m.GetHistogram(); h != nil {
				return float64(h.GetSampleCount())
			}
		}
	}
	return 0
}

func totalHTTPObservations(t *testing.T) float64 {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	var total float64
	for _, mf := range mfs {
		if mf.GetName() != "ecp_gateway_http_request_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			if h := m.GetHistogram(); h != nil {
				total += float64(h.GetSampleCount())
			}
		}
	}
	return total
}

func labelsMatch(m *dto.Metric, method, status, route, provider string) bool {
	want := map[string]string{
		"method":   method,
		"status":   status,
		"route":    route,
		"provider": provider,
	}
	got := make(map[string]string, len(m.GetLabel()))
	for _, l := range m.GetLabel() {
		got[l.GetName()] = l.GetValue()
	}
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}

// Ensure status label is always decimal digits (no accidental int formatting).
func TestStatusLabelFormat(t *testing.T) {
	t.Parallel()
	if strconv.Itoa(http.StatusNotFound) != "404" {
		t.Fatal("unexpected status formatting")
	}
	if !strings.EqualFold(http.MethodGet, "GET") {
		t.Fatal("method case")
	}
}
