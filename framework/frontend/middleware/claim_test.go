package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	kresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

// Shared test fixtures for provider/resource/verb/subject/path-key literals used across
// this package's tests.
const (
	testProviderCompute   = "seca.compute"
	testBaseSecaCompute   = "/providers/seca.compute"
	testResourceInstances = "instances"
	testVerbGet           = "get"
	testVerbList          = "list"
	testSubjectAlice      = "alice"
	pathKeyTenant         = "tenant"
	pathKeyName           = "name"
	pathKeyWorkspace      = "workspace"
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
			base:         testBaseSecaCompute,
			pattern:      "GET /providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances",
			pathValues:   map[string]string{pathKeyTenant: "t1", pathKeyWorkspace: "w1"},
			method:       http.MethodGet,
			wantResource: testResourceInstances,
			wantVerb:     testVerbList,
		},
		{
			name:         "GET item → get",
			base:         testBaseSecaCompute,
			pattern:      "GET /providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}",
			pathValues:   map[string]string{pathKeyTenant: "t1", pathKeyWorkspace: "w1", pathKeyName: "inst1"},
			method:       http.MethodGet,
			wantResource: testResourceInstances,
			wantVerb:     testVerbGet,
		},
		{
			name:         "PUT → put",
			base:         testBaseSecaCompute,
			pattern:      "PUT /providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}",
			pathValues:   map[string]string{pathKeyTenant: "t1", pathKeyWorkspace: "w1", pathKeyName: "inst1"},
			method:       http.MethodPut,
			wantResource: testResourceInstances,
			wantVerb:     "put",
		},
		{
			name:         "DELETE → delete",
			base:         testBaseSecaCompute,
			pattern:      "DELETE /providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}",
			pathValues:   map[string]string{pathKeyTenant: "t1", pathKeyWorkspace: "w1", pathKeyName: "inst1"},
			method:       http.MethodDelete,
			wantResource: testResourceInstances,
			wantVerb:     "delete",
		},
		{
			name:         "POST action → post.<action>",
			base:         testBaseSecaCompute,
			pattern:      "POST /providers/seca.compute/v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}/restart",
			pathValues:   map[string]string{pathKeyTenant: "t1", pathKeyWorkspace: "w1", pathKeyName: "inst1"},
			method:       http.MethodPost,
			wantResource: testResourceInstances,
			wantVerb:     "post.restart",
		},
		{
			name:         "GET tenant-scoped collection (no workspace) → list",
			base:         testBaseSecaCompute,
			pattern:      "GET /providers/seca.compute/v1/tenants/{tenant}/skus",
			pathValues:   map[string]string{pathKeyTenant: "t1"},
			method:       http.MethodGet,
			wantResource: "skus",
			wantVerb:     testVerbList,
		},
		{
			name:         "GET tenant-scoped item (no workspace) → get",
			base:         testBaseSecaCompute,
			pattern:      "GET /providers/seca.compute/v1/tenants/{tenant}/skus/{name}",
			pathValues:   map[string]string{pathKeyTenant: "t1", pathKeyName: "sku1"},
			method:       http.MethodGet,
			wantResource: "skus",
			wantVerb:     testVerbGet,
		},
		{
			name:         "nested resource collection → list",
			base:         "/providers/seca.network",
			pattern:      "GET /providers/seca.network/v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/subnets",
			pathValues:   map[string]string{pathKeyTenant: "t1", pathKeyWorkspace: "w1", "network": "net1"},
			method:       http.MethodGet,
			wantResource: "networks/subnets",
			wantVerb:     testVerbList,
		},
		{
			name:         "nested resource item → get",
			base:         "/providers/seca.network",
			pattern:      "GET /providers/seca.network/v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/subnets/{name}",
			pathValues:   map[string]string{pathKeyTenant: "t1", pathKeyWorkspace: "w1", "network": "net1", pathKeyName: "sub1"},
			method:       http.MethodGet,
			wantResource: "networks/subnets",
			wantVerb:     testVerbGet,
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

// TestSECAClaimExtractor_Region verifies that the region passed to the constructor —
// not a process-global — ends up on claim.Region, and that a global-server extractor
// (constructed with an empty region) always produces an empty claim.Region.
func TestSECAClaimExtractor_Region(t *testing.T) {
	t.Parallel()

	r := newPatternRequest(http.MethodGet, "/", "GET "+testBaseSecaCompute+"/v1/tenants/{tenant}/workspaces/{workspace}/instances",
		map[string]string{pathKeyTenant: "t1", pathKeyWorkspace: "w1"})

	regional := SECAClaimExtractor(testProviderCompute, testBaseSecaCompute, "r1")
	claim, err := regional(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claim.Region != "r1" {
		t.Errorf("Region = %q, want %q", claim.Region, "r1")
	}

	global := SECAClaimExtractor(testProviderCompute, testBaseSecaCompute, "")
	claim, err = global(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claim.Region != "" {
		t.Errorf("Region = %q, want empty (global server)", claim.Region)
	}
}

func TestResourceAndVerb_NetworkBase(t *testing.T) {
	t.Parallel()
	r := newPatternRequest(http.MethodGet, "/", "GET /providers/seca.network/v1/tenants/{tenant}/workspaces/{workspace}/networks/{network}/route-tables/{name}", map[string]string{
		pathKeyTenant: "t1", pathKeyWorkspace: "w1", "network": "net1", pathKeyName: "rt1",
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
	r := httptest.NewRequestWithContext(context.Background(), method, url, nil)
	// r.Pattern is normally set by http.ServeMux; replicate it for unit tests.
	r.Pattern = pattern
	for k, v := range pathValues {
		r.SetPathValue(k, v)
	}
	return r
}

// TestSECAClaimExtractor_RequestRegion verifies that on a gateway serving several regions
// the claim carries the region the request was addressed to, not the process default — so a
// region-scoped RoleAssignment denies a caller outside its regions.
func TestSECAClaimExtractor_RequestRegion(t *testing.T) {
	t.Parallel()

	r := newPatternRequest(http.MethodGet, "/", "GET "+testBaseSecaCompute+"/v1/tenants/{tenant}/workspaces/{workspace}/instances",
		map[string]string{pathKeyTenant: "t1", pathKeyWorkspace: "w1"})
	r = r.WithContext(kresource.ContextWithRegion(r.Context(), "r2"))

	claim, err := SECAClaimExtractor(testProviderCompute, testBaseSecaCompute, "r1")(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claim.Region != "r2" {
		t.Errorf("Region = %q, want %q (the region the request was addressed to)", claim.Region, "r2")
	}
}
