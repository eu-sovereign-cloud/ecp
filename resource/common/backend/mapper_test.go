package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
)

func TestExtractSegment(t *testing.T) {
	testCases := []struct {
		name     string
		resource string
		segment  string
		want     string
	}{
		// Reference.resource path with tenant/workspace scope:
		// tenants/{tenant}/workspaces/{workspace}/{collection}/{name}
		// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
		{"segment at the beginning", "workspaces/ws-1/block-storages/my-storage", "workspaces/", "ws-1"},
		{"segment in the middle", "seca.storage/v1/tenants/t-1/workspaces/ws-1/skus/s", "workspaces/", "ws-1"},
		{"segment at the end", "tenants/t-1/workspaces/ws-1", "workspaces/", "ws-1"},
		{"no segment found", "block-storages/my-storage", "workspaces/", ""},
		{"empty resource string", "", "workspaces/", ""},
		// A path boundary is required, so a collection whose name ends in the segment
		// must not match: "workspaces/" is not the "sub-workspaces/" prefix.
		{"segment is a suffix of another path element", "sub-workspaces/ws-1", "workspaces/", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, extractSegment(tc.resource, tc.segment))
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
