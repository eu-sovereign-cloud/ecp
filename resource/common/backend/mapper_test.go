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

// TestReferenceRoundTripIsVerbatim pins the property a client depends on: a reference is
// handed back exactly as it was written, in whichever of the two representations the spec
// allows the client picked. Rewriting one into the other made reads disagree with writes -
// a terraform apply that sent {tenant, resource} got {resource: "tenants/.../..."} back and
// failed with "produced an unexpected new value".
// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
func TestReferenceRoundTripIsVerbatim(t *testing.T) {
	testCases := []struct {
		name string
		ref  schemav1.Reference
	}{
		{
			name: "scope as fields",
			ref:  schemav1.Reference{Tenant: "t-1", Workspace: "ws-1", Resource: "block-storages/my-storage"},
		},
		{
			name: "scope as fields, nested path",
			ref:  schemav1.Reference{Tenant: "t-1", Workspace: "ws-1", Resource: "networks/n1/route-tables/rt1"},
		},
		{
			// A tenant-scoped sku referenced from a workspace-scoped resource: the tenant
			// cannot be inferred from context, so this is the shape that used to be rewritten.
			name: "tenant field with a provider field",
			ref:  schemav1.Reference{Provider: "seca.network/v1", Tenant: "t-1", Resource: "skus/fast-local"},
		},
		{
			// A client that holds only the target's URN sends the whole URN as the path.
			name: "scope spelled out in the path",
			ref:  schemav1.Reference{Resource: "seca.network/v1/tenants/t-1/workspaces/ws-1/networks/n1/route-tables/rt1"},
		},
		{
			name: "no scope at all - inferred from context",
			ref:  schemav1.Reference{Resource: "skus/fast-local"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.ref, ReferenceToCR(ReferenceFromCR(tc.ref)))
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
