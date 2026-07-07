//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	authv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
)

// TestBench fires a configurable number of authenticated requests against the
// deployed gateway to populate the Prometheus auth latency histograms.
//
// Gate: set E2E_BENCH=1 to run; the test is skipped otherwise so it does not
// accidentally inflate the metrics during a normal test run.
// Tune: E2E_BENCH_REQUESTS=N controls the number of requests (default 500).
//
// Typical use:
//
//	E2E_BENCH=1 E2E_BENCH_REQUESTS=500 go test -v -tags=integration \
//	    -run=TestBench ./test/integration/gateway-global/
func TestBench(t *testing.T) {
	if os.Getenv("E2E_BENCH") != "1" {
		t.Skip("E2E_BENCH=1 required; set to run the load workload")
	}
	if !authEnabled() {
		t.Skip("E2E_AUTH_ENABLED=false: benchmark requires auth middleware to be active")
	}

	n := 500
	if v := os.Getenv("E2E_BENCH_REQUESTS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			n = parsed
		}
	}

	regionURL := fmt.Sprintf("http://localhost:%d/providers/seca.region", globalLocalPort)
	authURL := fmt.Sprintf("http://localhost:%d/providers/seca.authorization", globalLocalPort)

	// Mix of allow (admin lists regions) and deny (nobody lists roles) requests to exercise
	// both authorization branches.
	allowClient, err := regionv1.NewClientWithResponses(regionURL, regionv1.WithRequestEditorFn(adminEditor()))
	if err != nil {
		t.Fatalf("create allow client: %v", err)
	}

	// nobody has valid credentials but no grant for seca.authorization → always 403 (deny path).
	// (ra-wildcard now grants region-viewer to everyone, so a region read is no longer a deny.)
	denyClient, err := authv1.NewClientWithResponses(
		authURL,
		authv1.WithRequestEditorFn(identityEditor("nobody", "nobody-pass")),
	)
	if err != nil {
		t.Fatalf("create deny client: %v", err)
	}

	allows, denies := 0, 0
	for i := range n {
		if i%4 == 0 {
			// 1 in 4 requests exercises the deny path.
			resp, err := denyClient.ListRolesWithResponse(context.Background(), testTenant, &authv1.ListRolesParams{})
			if err != nil {
				t.Fatalf("deny request %d: %v", i, err)
			}
			_ = resp
			denies++
		} else {
			resp, err := allowClient.ListRegionsWithResponse(context.Background(), &regionv1.ListRegionsParams{})
			if err != nil {
				t.Fatalf("allow request %d: %v", i, err)
			}
			_ = resp
			allows++
		}
	}
	t.Logf("Bench complete: %d allow + %d deny = %d total requests", allows, denies, n)
}
