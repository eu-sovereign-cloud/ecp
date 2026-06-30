package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestResourceAndVerb verifies the route-pattern parser that derives (resource,verb)
// from an HTTP request matched by Go 1.22+ http.ServeMux.
//
// The parser reads r.Pattern (set by the mux after matching), strips the provider
// base URL and the tenant/workspace anchors, then collects only the static path
// segments to form the resource kind path.
func TestResourceAndVerb(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		base         string // provider base URL baked into the extractor
		pattern      string // full mux pattern (method + path)
		pathValues   map[string]string
		method       string
		wantResource string
		wantVerb     string
	}{
		{
			name:         "GET collection → list",
			base:         "/providers/seca.compute",
			pattern:      "GET /providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances",
			pathValues:   map[string]string{"tenant": "t1", "workspace": "w1"},
			method:       http.MethodGet,
			wantResource: "instances",
			wantVerb:     "list",
		},
		{
			name:         "GET item → get",
			base:         "/providers/seca.compute",
			pattern:      "GET /providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}",
			pathValues:   map[string]string{"tenant": "t1", "workspace": "w1", "name": "inst1"},
			method:       http.MethodGet,
			wantResource: "instances",
			wantVerb:     "get",
		},
		{
			name:         "PUT → put",
			base:         "/providers/seca.compute",
			pattern:      "PUT /providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}",
			pathValues:   map[string]string{"tenant": "t1", "workspace": "w1", "name": "inst1"},
			method:       http.MethodPut,
			wantResource: "instances",
			wantVerb:     "put",
		},
		{
			name:         "DELETE → delete",
			base:         "/providers/seca.compute",
			pattern:      "DELETE /providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}",
			pathValues:   map[string]string{"tenant": "t1", "workspace": "w1", "name": "inst1"},
			method:       http.MethodDelete,
			wantResource: "instances",
			wantVerb:     "delete",
		},
		{
			name:         "POST action → post.<action>",
			base:         "/providers/seca.compute",
			pattern:      "POST /providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}/restart",
			pathValues:   map[string]string{"tenant": "t1", "workspace": "w1", "name": "inst1"},
			method:       http.MethodPost,
			wantResource: "instances",
			wantVerb:     "post.restart",
		},
		{
			name:         "GET tenant-scoped collection (no workspace) → list",
			base:         "/providers/seca.compute",
			pattern:      "GET /providers/seca.compute/v1/tenants/{tenant}/skus",
			pathValues:   map[string]string{"tenant": "t1"},
			method:       http.MethodGet,
			wantResource: "skus",
			wantVerb:     "list",
		},
		{
			name:         "GET tenant-scoped item (no workspace) → get",
			base:         "/providers/seca.compute",
			pattern:      "GET /providers/seca.compute/v1/tenants/{tenant}/skus/{name}",
			pathValues:   map[string]string{"tenant": "t1", "name": "sku1"},
			method:       http.MethodGet,
			wantResource: "skus",
			wantVerb:     "get",
		},
		{
			name:         "nested resource collection → list",
			base:         "/providers/seca.network",
			pattern:      "GET /providers/seca.network/v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/subnets",
			pathValues:   map[string]string{"tenant": "t1", "workspace": "w1", "network": "net1"},
			method:       http.MethodGet,
			wantResource: "networks/subnets",
			wantVerb:     "list",
		},
		{
			name:         "nested resource item → get",
			base:         "/providers/seca.network",
			pattern:      "GET /providers/seca.network/v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/subnets/{name}",
			pathValues:   map[string]string{"tenant": "t1", "workspace": "w1", "network": "net1", "name": "sub1"},
			method:       http.MethodGet,
			wantResource: "networks/subnets",
			wantVerb:     "get",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newPatternRequest(tc.method, "/", tc.pattern, tc.pathValues)

			name := tc.pathValues["name"]
			gotResource, gotVerb, err := resourceAndVerb(r, tc.base, name)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotResource != tc.wantResource {
				t.Errorf("resource = %q, want %q", gotResource, tc.wantResource)
			}
			if gotVerb != tc.wantVerb {
				t.Errorf("verb = %q, want %q", gotVerb, tc.wantVerb)
			}
		})
	}
}

func TestResourceAndVerb_NetworkBase(t *testing.T) {
	t.Parallel()
	r := newPatternRequest(http.MethodGet, "/", "GET /providers/seca.network/v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/route-tables/{name}", map[string]string{
		"tenant": "t1", "workspace": "w1", "network": "net1", "name": "rt1",
	})
	resource, verb, err := resourceAndVerb(r, "/providers/seca.network", "rt1")
	if err != nil {
		t.Fatal(err)
	}
	if resource != "networks/route-tables" {
		t.Errorf("resource = %q, want %q", resource, "networks/route-tables")
	}
	if verb != "get" {
		t.Errorf("verb = %q, want %q", verb, "get")
	}
}

// newPatternRequest creates an *http.Request with r.Pattern and path values set,
// mimicking what the Go 1.22+ mux does after a successful route match.
func newPatternRequest(method, url, pattern string, pathValues map[string]string) *http.Request {
	r := httptest.NewRequest(method, url, nil)
	// r.Pattern is normally set by http.ServeMux; replicate it for unit tests.
	r.Pattern = pattern
	for k, v := range pathValues {
		r.SetPathValue(k, v)
	}
	return r
}
