package backend

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

const workspaceSegment = "workspaces/"

func TestExtractSegment(t *testing.T) {
	testCases := []struct {
		name     string
		resource string
		segment  string
		expected string
	}{
		{
			name:     "segment at the beginning",
			resource: "workspaces/ws-1/block-storages/my-storage",
			segment:  workspaceSegment,
			expected: "ws-1",
		},
		{
			name:     "segment in the middle",
			resource: "tenants/t-1/workspaces/ws-1/block-storages/my-storage",
			segment:  workspaceSegment,
			expected: "ws-1",
		},
		{
			name:     "segment at the end",
			resource: "tenants/t-1/workspaces/ws-1",
			segment:  workspaceSegment,
			expected: "ws-1",
		},
		{
			name:     "no segment found",
			resource: "block-storages/my-storage",
			segment:  workspaceSegment,
			expected: "",
		},
		{
			name:     "empty resource string",
			resource: "",
			segment:  workspaceSegment,
			expected: "",
		},
		{
			name:     "segment is a suffix of another path element",
			resource: "sub-workspaces/ws-1",
			segment:  workspaceSegment,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, extractSegment(tc.resource, tc.segment))
		})
	}
}

func TestReferenceRoundTripPreservesRepresentation(t *testing.T) {
	refs := []schemav1.Reference{
		{
			Provider:  "seca.network/v1",
			Resource:  "networks/poc-net/subnets/poc-subnet",
			Tenant:    "test-tenant",
			Workspace: "poc",
		},
		{
			Provider: "seca.network/v1",
			Resource: "seca.network/v1/tenants/test-tenant/workspaces/poc/networks/poc-net/subnets/poc-subnet",
		},
	}

	for _, want := range refs {
		got := ReferenceToCR(ReferenceFromCR(want))
		assert.Equal(t, want, got)
	}
}

// FuzzExtractSegment verifies that extractSegment never panics on arbitrary input.
func FuzzExtractSegment(f *testing.F) {
	f.Add("workspaces/ws-1/block-storages/my-storage", workspaceSegment)
	f.Add("tenants/t-1/workspaces/ws-1", workspaceSegment)
	f.Add("workspaces/ws-1", workspaceSegment)
	f.Add("providers/ionos/regions/de-fra", "regions/")
	f.Add("", "workspaces/")
	f.Add("/", "/")
	f.Add("a/b/c", "b/")
	// long paths around Kubernetes' 253-char DNS subdomain limit
	f.Add(strings.Repeat("a", 253)+"/workspaces/ws-1", "workspaces/")
	f.Add(strings.Repeat("a", 254)+"/workspaces/ws-1", "workspaces/")
	f.Add("workspaces/"+strings.Repeat("b", 64), "workspaces/")

	f.Fuzz(func(t *testing.T, resource, segment string) {
		extractSegment(resource, segment)
	})
}

func TestParseReference(t *testing.T) {
	ref := domain.Reference{Resource: "seca.network/v1/tenants/test-tenant/workspaces/poc/networks/poc-net/subnets/poc-subnet"}
	assert.Equal(t, ReferenceTarget{
		Tenant:    "test-tenant",
		Workspace: "poc",
		Name:      "poc-subnet",
	}, ParseReference(ref, "fallback"))
}
