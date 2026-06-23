package backend

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
			name:              "multiple segments present",
			resource:          "providers/ionos/regions/de-fra/tenants/t-1/workspaces/ws-1/block-storages/my-storage",
			segment:           "workspaces/",
			expectedValue:     "ws-1",
			expectedRemaining: "providers/ionos/regions/de-fra/tenants/t-1/block-storages/my-storage",
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
