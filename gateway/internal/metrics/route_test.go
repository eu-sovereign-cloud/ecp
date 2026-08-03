package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		wantRoute    string
		wantProvider string
	}{
		{
			name:         "workspace get",
			path:         "/providers/seca.workspace/v1/tenants/acme/workspaces/ws-1",
			wantRoute:    "/providers/seca.workspace/v1/tenants/{tenant}/workspaces/{name}",
			wantProvider: "seca.workspace",
		},
		{
			name:         "workspace list",
			path:         "/providers/seca.workspace/v1/tenants/acme/workspaces",
			wantRoute:    "/providers/seca.workspace/v1/tenants/{tenant}/workspaces",
			wantProvider: "seca.workspace",
		},
		{
			name:         "nested network subnet",
			path:         "/providers/seca.network/v1/tenants/t1/workspaces/w1/networks/n1/subnets/s1",
			wantRoute:    "/providers/seca.network/v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/subnets/{name}",
			wantProvider: "seca.network",
		},
		{
			name:         "instance under workspace",
			path:         "/providers/seca.compute/v1/tenants/t1/workspaces/w1/instances/i1",
			wantRoute:    "/providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}",
			wantProvider: "seca.compute",
		},
		{
			name:         "region list",
			path:         "/providers/seca.region/v1/regions",
			wantRoute:    "/providers/seca.region/v1/regions",
			wantProvider: "seca.region",
		},
		{
			name:         "region get",
			path:         "/providers/seca.region/v1/regions/eu-central-1",
			wantRoute:    "/providers/seca.region/v1/regions/{name}",
			wantProvider: "seca.region",
		},
		{
			name:         "healthz",
			path:         "/healthz",
			wantRoute:    "/healthz",
			wantProvider: "system",
		},
		{
			name:         "readyz",
			path:         "/readyz",
			wantRoute:    "/readyz",
			wantProvider: "system",
		},
		{
			name:         "unknown path",
			path:         "/api/v1/foo",
			wantRoute:    "{other}",
			wantProvider: "unknown",
		},
		{
			name:         "empty path",
			path:         "",
			wantRoute:    "{other}",
			wantProvider: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotRoute, gotProvider := normalizePath(tt.path)
			if gotRoute != tt.wantRoute {
				t.Errorf("route = %q, want %q", gotRoute, tt.wantRoute)
			}
			if gotProvider != tt.wantProvider {
				t.Errorf("provider = %q, want %q", gotProvider, tt.wantProvider)
			}
		})
	}
}

func TestStripMethodPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"GET /providers/seca.region/v1/regions", "/providers/seca.region/v1/regions"},
		{"PUT /providers/seca.workspace/v1/tenants/{tenant}/workspaces/{name}", "/providers/seca.workspace/v1/tenants/{tenant}/workspaces/{name}"},
		{"/already/stripped", "/already/stripped"},
		{"NOTAMETHOD /path", "NOTAMETHOD /path"},
	}
	for _, tt := range tests {
		if got := stripMethodPrefix(tt.in); got != tt.want {
			t.Errorf("stripMethodPrefix(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRouteFromRequest_PrefersPattern(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /providers/seca.workspace/v1/tenants/{tenant}/workspaces/{name}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	var capturedRoute, capturedProvider string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
		capturedRoute, capturedProvider = routeFromRequest(r)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/providers/seca.workspace/v1/tenants/acme/workspaces/ws-1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	want := "/providers/seca.workspace/v1/tenants/{tenant}/workspaces/{name}"
	if capturedRoute != want {
		t.Errorf("route = %q, want %q (pattern=%q)", capturedRoute, want, req.Pattern)
	}
	if capturedProvider != "seca.workspace" {
		t.Errorf("provider = %q, want seca.workspace", capturedProvider)
	}
}

func TestRouteFromRequest_FallbackWithoutPattern(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/providers/seca.compute/v1/tenants/t/workspaces/w/instances/i", nil)
	// Pattern left empty — simulate no ServeMux match bookkeeping.
	route, provider := routeFromRequest(req)
	want := "/providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}"
	if route != want {
		t.Errorf("route = %q, want %q", route, want)
	}
	if provider != "seca.compute" {
		t.Errorf("provider = %q, want seca.compute", provider)
	}
}
