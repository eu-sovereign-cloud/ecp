package cmd

import (
	"fmt"
	"slices"
	"strings"
)

// resolveRegions returns the set of regions the regional gateway serves and the region an
// unprefixed request falls back to.
//
// One region (--region, the only form before multi-region support) keeps behaving exactly as
// it did: it is both the served set and the default. --regions widens the served set; --region
// then names which of them an unprefixed request means, and defaults to the first listed.
func resolveRegions(region string, regions []string) (served []string, defaultRegion string, err error) {
	defaultRegion = strings.TrimSpace(region)

	for _, r := range regions {
		if r = strings.TrimSpace(r); r != "" && !slices.Contains(served, r) {
			served = append(served, r)
		}
	}

	switch {
	case len(served) == 0 && defaultRegion == "":
		// Fail fast: no region mis-scopes every regional request (authz region, resource
		// placement, list filtering) for the process life.
		return nil, "", fmt.Errorf("region is required: set --region/--regions or the REGION/REGIONS environment variable")
	case len(served) == 0:
		served = []string{defaultRegion}
	case defaultRegion == "":
		defaultRegion = served[0]
	case !slices.Contains(served, defaultRegion):
		return nil, "", fmt.Errorf("--region %q is not listed in --regions %v", defaultRegion, served)
	}

	return served, defaultRegion, nil
}
