package backend

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
)

func TestExtractAndStripSegment(t *testing.T) {
	testCases := []struct {
		name              string
		resource          string
		segment           string
		expectedValue     string
		expectedRemaining string
	}{
		{
			name:              "segment at the beginning",
			resource:          "workspaces/ws-1/block-storages/my-storage",
			segment:           "workspaces/",
			expectedValue:     "ws-1",
			expectedRemaining: "block-storages/my-storage",
		},
		{
			name:              "segment in the middle",
			resource:          "tenants/t-1/workspaces/ws-1/block-storages/my-storage",
			segment:           "workspaces/",
			expectedValue:     "ws-1",
			expectedRemaining: "tenants/t-1/block-storages/my-storage",
		},
		{
			name:              "segment at the end",
			resource:          "tenants/t-1/workspaces/ws-1",
			segment:           "workspaces/",
			expectedValue:     "ws-1",
			expectedRemaining: "tenants/t-1",
		},
		{
			name:              "segment is the only component",
			resource:          "workspaces/ws-1",
			segment:           "workspaces/",
			expectedValue:     "ws-1",
			expectedRemaining: "",
		},
		{
			name:              "no segment found",
			resource:          "block-storages/my-storage",
			segment:           "workspaces/",
			expectedValue:     "",
			expectedRemaining: "",
		},
		{
			name:              "empty resource string",
			resource:          "",
			segment:           "workspaces/",
			expectedValue:     "",
			expectedRemaining: "",
		},
		{
			name: "multiple segments present",
			// Reference.resource path with tenant/workspace scope:
			// tenants/{tenant}/workspaces/{workspace}/{collection}/{name}
			// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
			resource:          "tenants/t-1/workspaces/ws-1/block-storages/my-storage",
			segment:           "workspaces/",
			expectedValue:     "ws-1",
			expectedRemaining: "tenants/t-1/block-storages/my-storage",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value, remaining := extractAndStripSegment(tc.resource, tc.segment)
			assert.Equal(t, tc.expectedValue, value)
			assert.Equal(t, tc.expectedRemaining, remaining)
		})
	}
}

// TestReferenceFromCRScopeOrder pins the URN segment order the SDK and the terraform provider
// rebuild the full URN from: scope first, then the (possibly nested) resource path.
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
func TestReferenceFromCRScopeOrder(t *testing.T) {
	testCases := []struct {
		name     string
		ref      schemav1.Reference
		expected string
	}{
		{
			name:     "flat resource",
			ref:      schemav1.Reference{Tenant: "t-1", Workspace: "ws-1", Resource: "block-storages/my-storage"},
			expected: "tenants/t-1/workspaces/ws-1/block-storages/my-storage",
		},
		{
			// The nested case that used to land as networks/n1/tenants/t-1/...
			name:     "network-scoped resource",
			ref:      schemav1.Reference{Tenant: "t-1", Workspace: "ws-1", Resource: "networks/n1/route-tables/rt1"},
			expected: "tenants/t-1/workspaces/ws-1/networks/n1/route-tables/rt1",
		},
		{
			name:     "tenant only",
			ref:      schemav1.Reference{Tenant: "t-1", Resource: "skus/fast-local"},
			expected: "tenants/t-1/skus/fast-local",
		},
		{
			// A client that holds only the target's URN sends the whole URN as the path,
			// provider pair included. The scope belongs after that pair, not before it.
			name:     "path opening with the provider URN",
			ref:      schemav1.Reference{Tenant: "t-1", Workspace: "ws-1", Resource: "seca.network/v1/networks/n1/route-tables/rt1"},
			expected: "seca.network/v1/tenants/t-1/workspaces/ws-1/networks/n1/route-tables/rt1",
		},
		{
			// A dot alone is not a provider pair: "v1" must follow, or the path is a plain
			// collection path and must be left alone.
			name:     "collection segment containing a dot is not a provider",
			ref:      schemav1.Reference{Tenant: "t-1", Resource: "my.things/thing-1"},
			expected: "tenants/t-1/my.things/thing-1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := ReferenceFromCR(tc.ref)
			assert.Equal(t, tc.expected, out.Resource)

			// The scope must be extracted back out unchanged, leaving the CR form untouched.
			assert.Equal(t, tc.ref, ReferenceToCR(out))
		})
	}
}

// FuzzExtractAndStripSegment verifies that extractAndStripSegment never panics on arbitrary input.
func FuzzExtractAndStripSegment(f *testing.F) {
	f.Add("workspaces/ws-1/block-storages/my-storage", "workspaces/")
	f.Add("tenants/t-1/workspaces/ws-1", "workspaces/")
	f.Add("workspaces/ws-1", "workspaces/")
	f.Add("providers/ionos/regions/de-fra", "regions/")
	f.Add("", "workspaces/")
	f.Add("/", "/")
	f.Add("a/b/c", "b/")
	// long paths around Kubernetes' 253-char DNS subdomain limit
	f.Add(strings.Repeat("a", 253)+"/workspaces/ws-1", "workspaces/")
	f.Add(strings.Repeat("a", 254)+"/workspaces/ws-1", "workspaces/")
	f.Add("workspaces/"+strings.Repeat("b", 64), "workspaces/")

	f.Fuzz(func(t *testing.T, resource, segment string) {
		extractAndStripSegment(resource, segment)
	})
}
