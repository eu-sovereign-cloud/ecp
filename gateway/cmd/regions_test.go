package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveRegions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		region        string
		regions       []string
		wantServed    []string
		wantDefault   string
		wantErrSubstr string
	}{
		{
			name: "one --region serves and defaults to it", region: " itbg-bergamo ",
			wantServed: []string{"itbg-bergamo"}, wantDefault: "itbg-bergamo",
		},
		{
			name: "--regions alone serves them all and defaults to the first", regions: []string{"a", "b"},
			wantServed: []string{"a", "b"}, wantDefault: "a",
		},
		{
			name: "a single --regions entry is also the default", regions: []string{"a"},
			wantServed: []string{"a"}, wantDefault: "a",
		},
		{
			name: "--region names the default among --regions", region: "b", regions: []string{"a", "b"},
			wantServed: []string{"a", "b"}, wantDefault: "b",
		},
		{
			name: "blanks and duplicates are dropped", regions: []string{"a", " ", "a", " b "},
			wantServed: []string{"a", "b"}, wantDefault: "a",
		},
		{
			name: "neither is set", wantErrSubstr: "region is required",
		},
		{
			name:   "a default outside the served set is a startup failure",
			region: "c", regions: []string{"a", "b"}, wantErrSubstr: "is not listed in",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			served, defaultRegion, err := resolveRegions(tc.region, tc.regions)
			if tc.wantErrSubstr != "" {
				require.ErrorContains(t, err, tc.wantErrSubstr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantServed, served)
			require.Equal(t, tc.wantDefault, defaultRegion)
		})
	}
}
