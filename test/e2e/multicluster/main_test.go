//go:build multicluster

// Package multicluster holds the two-cluster end-to-end suite: the global
// gateway runs in one cluster, the regional gateway and delegator in another,
// and the only thing joining them is the Region CR the global gateway
// advertises.
//
// The suite is deliberately given ONE endpoint — the global gateway — plus a
// kubeconfig context for each cluster. It never learns the regional address by
// configuration; it must read it off the region catalog, which is what makes
// this a topology test rather than a fixture check. See
// test/internal/scripts/register-region.sh for the join step.
//
// Kept behind its own build tag so the single-cluster `make kind-e2e` stays
// fast; run it with `make kind-multicluster-e2e`.
package multicluster

import (
	"fmt"
	"log"
	"os"
	"testing"

	"k8s.io/client-go/dynamic"

	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"

	authhelper "github.com/eu-sovereign-cloud/ecp/test/internal/authhelper"
	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

const (
	globalLabel = "app=gateway-global"

	// testRegion must match both the regional gateway's REGION env and the name
	// register-region.sh writes.
	testRegion    = "itbg-bergamo"
	testWorkspace = "mc-workspace"
)

var (
	systemNamespace = envOr("SYSTEM_NAMESPACE", "e2e-ecp")
	testTenant      = envOr("E2E_TENANT", "test-tenant")

	globalContext   = envOr("MULTICLUSTER_GLOBAL_CONTEXT", "kind-e2e-global")
	regionalContext = envOr("MULTICLUSTER_REGIONAL_CONTEXT", "kind-e2e-regional")
)

var (
	// regionClient is the ONLY pre-built API client: everything regional is
	// constructed at run time from the URLs this catalog advertises.
	regionClient *regionv1.ClientWithResponses

	// Dynamic clients for both clusters, used to assert which side of the
	// boundary a resource actually landed on.
	globalDyn   dynamic.Interface
	regionalDyn dynamic.Interface

	// authEditor signs requests for whichever auth plugin the stack was
	// deployed with; both gateways run the same one.
	authEditor = authhelper.AdminEditor()
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func TestMain(m *testing.M) {
	globalCfg, globalClientset, err := testenv.SetupK8sClientForContext(globalContext)
	if err != nil {
		log.Fatalf("Failed to set up k8s client for global context %q: %v", globalContext, err)
	}
	regionalCfg, err := testenv.RestConfigForContext(regionalContext)
	if err != nil {
		log.Fatalf("Failed to set up k8s client for regional context %q: %v", regionalContext, err)
	}

	if globalDyn, err = dynamic.NewForConfig(globalCfg); err != nil {
		log.Fatalf("Failed to create global dynamic client: %v", err)
	}
	if regionalDyn, err = dynamic.NewForConfig(regionalCfg); err != nil {
		log.Fatalf("Failed to create regional dynamic client: %v", err)
	}

	// Only the global gateway is port-forwarded. The regional one must be
	// reached at its advertised address or not at all.
	globalPF, err := testenv.StartPortForward(globalClientset, globalCfg, systemNamespace, globalLabel)
	if err != nil {
		log.Fatalf("Failed to port-forward to global gateway: %v", err)
	}

	globalURL := fmt.Sprintf("http://localhost:%d", globalPF.LocalPort)
	if regionClient, err = regionv1.NewClientWithResponses(
		globalURL+"/providers/seca.region",
		regionv1.WithRequestEditorFn(authEditor),
	); err != nil {
		globalPF.Close()
		log.Fatalf("Failed to create region SDK client: %v", err)
	}

	log.Printf("Multicluster environment ready (global=%s regional=%s). Running tests...", globalContext, regionalContext)
	code := m.Run()

	globalPF.Close()
	os.Exit(code)
}
