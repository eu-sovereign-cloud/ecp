package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	// normalizePath only runs for requests the ServeMux did not match, so it never
	// needs to reproduce a real route template: it keeps the allowlisted provider
	// and version literals and collapses the caller-controlled remainder.
	tests := []struct {
		name         string
		path         string
		wantRoute    string
		wantProvider string
	}{
		{
			name:         "known provider and version",
			path:         "/providers/seca.workspace/v1/tenants/acme/workspaces/ws-1",
			wantRoute:    "/providers/seca.workspace/v1/{other}",
			wantProvider: "seca.workspace",
		},
		{
			name:         "known provider beta version",
			path:         "/providers/seca.network/v1beta1/tenants/t1/does-not-exist",
			wantRoute:    "/providers/seca.network/v1beta1/{other}",
			wantProvider: "seca.network",
		},
		{
			name:         "unknown provider is not preserved",
			path:         "/providers/seca.bogus/v1/tenants/x",
			wantRoute:    "{other}",
			wantProvider: "unknown",
		},
		{
			name:         "provider-shaped random segment is not preserved",
			path:         "/providers/seca." + strings.Repeat("a", 32) + "/v1/tenants/x",
			wantRoute:    "{other}",
			wantProvider: "unknown",
		},
		{
			name:         "version prefix is not enough",
			path:         "/providers/seca.compute/v1bogus/tenants/t1/workspaces/w1/instances/i1",
			wantRoute:    "{other}",
			wantProvider: "seca.compute",
		},
		{
			name:         "v2 prefix is not a supported version",
			path:         "/providers/seca.compute/v2/tenants/t1",
			wantRoute:    "{other}",
			wantProvider: "seca.compute",
		},
		{
			name:         "provider without version",
			path:         "/providers/seca.region",
			wantRoute:    "{other}",
			wantProvider: "seca.region",
		},
		{
			name:         "deep path does not grow the route label",
			path:         "/providers/seca.region/v1/" + strings.Repeat("x/", 64),
			wantRoute:    "/providers/seca.region/v1/{other}",
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

// TestNormalizePath_BoundedCardinality asserts the fallback cannot be driven to
// emit unbounded label values: sweeping providers, versions and path shapes must
// stay inside the closed set derived from the allowlists.
func TestNormalizePath_BoundedCardinality(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{routeOther: {}}
	for provider := range secaProviders {
		for version := range apiVersions {
			allowed["/providers/"+provider+"/"+version+"/"+routeOther] = struct{}{}
		}
	}
	for path := range systemPaths {
		allowed[path] = struct{}{}
	}

	allowedProviders := map[string]struct{}{"unknown": {}, "system": {}}
	for provider := range secaProviders {
		allowedProviders[provider] = struct{}{}
	}

	providers := []string{"seca.compute", "seca.evil", "seca.", "seca.compute-x", "notaprovider"}
	versions := []string{"v1", "v1beta1", "v1x", "v2", "v20", "", "V1"}
	suffixes := []string{"", "/tenants", "/tenants/t1/workspaces/w1/instances/i1", "/" + strings.Repeat("a/", 40)}

	for _, provider := range providers {
		for _, version := range versions {
			for _, suffix := range suffixes {
				path := "/providers/" + provider + "/" + version + suffix
				route, gotProvider := normalizePath(path)
				if _, ok := allowed[route]; !ok {
					t.Errorf("normalizePath(%q) route = %q, outside the closed route set", path, route)
				}
				if _, ok := allowedProviders[gotProvider]; !ok {
					t.Errorf("normalizePath(%q) provider = %q, outside the closed provider set", path, gotProvider)
				}
			}
		}
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
	// Pattern left empty — simulate no ServeMux match bookkeeping. Without a
	// matched template only the allowlisted prefix survives.
	route, provider := routeFromRequest(req)
	want := "/providers/seca.compute/v1/" + routeOther
	if route != want {
		t.Errorf("route = %q, want %q", route, want)
	}
	if provider != "seca.compute" {
		t.Errorf("provider = %q, want seca.compute", provider)
	}
}
