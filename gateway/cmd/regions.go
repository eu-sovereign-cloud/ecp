package cmd

import (
	"errors"
	"os"
	"slices"
	"strings"
)

// resolveRegions returns the set of regions the regional gateway serves and the region an
// unprefixed request is served as.
//
// The set is --regions, or the REGIONS environment variable when the flag is unset, trimmed
// and with blanks and duplicates dropped. The default is always the first entry, so a single
// region is both the whole served set and the default, and a list is ordered by intent.
func resolveRegions(regions []string) (served []string, defaultRegion string, err error) {
	if len(regions) == 0 {
		regions = strings.Split(os.Getenv("REGIONS"), ",")
	}

	for _, r := range regions {
		if r = strings.TrimSpace(r); r != "" && !slices.Contains(served, r) {
			served = append(served, r)
		}
	}

	if len(served) == 0 {
		// Fail fast: no region mis-scopes every regional request (authz region, resource
		// placement, list filtering) for the process life.
		return nil, "", errors.New("regions is required: set --regions or the REGIONS environment variable")
	}

	return served, served[0], nil
}
