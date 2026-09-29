package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResolveRegions is not parallel: every case pins REGIONS with t.Setenv, which the
// flag-unset cases fall back to.
func TestResolveRegions(t *testing.T) {
	tests := []struct {
		name          string
		regions       []string
		env           string
		wantServed    []string
		wantDefault   string
		wantErrSubstr string
	}{
		{
			name: "a single region serves and defaults to it", regions: []string{" itbg-bergamo "},
			wantServed: []string{"itbg-bergamo"}, wantDefault: "itbg-bergamo",
		},
		{
			name: "several regions serve them all and default to the first", regions: []string{"a", "b"},
			wantServed: []string{"a", "b"}, wantDefault: "a",
		},
		{
			name: "the order picks the default", regions: []string{"b", "a"},
			wantServed: []string{"b", "a"}, wantDefault: "b",
		},
		{
			name: "blanks and duplicates are dropped before the default is picked", regions: []string{" ", "a", "a", " b "},
			wantServed: []string{"a", "b"}, wantDefault: "a",
		},
		{
			name: "REGIONS is read when the flag is unset", env: "x, y",
			wantServed: []string{"x", "y"}, wantDefault: "x",
		},
		{
			name: "the flag wins over REGIONS", regions: []string{"a"}, env: "x,y",
			wantServed: []string{"a"}, wantDefault: "a",
		},
		{
			name: "an empty flag falls back to REGIONS", regions: []string{}, env: "x",
			wantServed: []string{"x"}, wantDefault: "x",
		},
		{
			name: "neither is set", wantErrSubstr: "regions is required",
		},
		{
			name: "only blanks is the same as unset", regions: []string{" ", ""}, env: " , ",
			wantErrSubstr: "regions is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("REGIONS", tc.env)
			served, defaultRegion, err := resolveRegions(tc.regions)
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

// TestResolveRegionsIgnoresREGION pins the removal of the single-region form: REGION is no
// longer a way to configure the gateway, so on its own it must fail startup rather than be
// quietly honoured.
func TestResolveRegionsIgnoresREGION(t *testing.T) {
	t.Setenv("REGIONS", "")
	t.Setenv("REGION", "itbg-bergamo")

	_, _, err := resolveRegions(nil)
	require.ErrorContains(t, err, "regions is required")
}

// TestRegionalFlags pins --regions as the one region flag: --region is gone, so an old
// invocation fails with cobra's unknown-flag error instead of being half-honoured.
func TestRegionalFlags(t *testing.T) {
	t.Parallel()

	require.Nil(t, regionalApiServerCMD.Flags().Lookup("region"))
	require.NotNil(t, regionalApiServerCMD.Flags().Lookup("regions"))
}
