//go:build e2e

// Package e2e holds the single end-to-end suite that exercises the whole ECP
// stack together: it drives the SECA REST API on the global and regional
// gateways and asserts that resources are reconciled all the way down to the
// delegator plugin (the dummy plugin by default). Unlike the integration suites
// — which test one component in isolation and never wait for reconciliation —
// this suite requires test-data, both gateways and the delegator to be deployed.
package e2e

import (
	"fmt"
	"log"
	"os"
	"testing"

	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	workspacev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"

	authhelper "github.com/eu-sovereign-cloud/ecp/test/internal/authhelper"

	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

const (
	regionalLabel = "app=gateway-regional"
	globalLabel   = "app=gateway-global"

	testWorkspace = "e2e-workspace"
	// testRegion is one of the regions provisioned by the test-data fixture and
	// the region the regional gateway is configured for.
	testRegion = "itbg-bergamo"
)

// systemNamespace (where the components run) and testTenant are overridable so the
// suite can run against a custom deployment. Defaults match the fixtures; keep
// SYSTEM_NAMESPACE / E2E_TENANT in sync with what the stack was deployed with (see
// test/Makefile).
var (
	systemNamespace = envOr("SYSTEM_NAMESPACE", "e2e-ecp")
	testTenant      = envOr("E2E_TENANT", "test-tenant")
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var (
	// Regional gateway (dummy authenticator) clients.
	storageClient   *storagev1.ClientWithResponses
	workspaceClient *workspacev1.ClientWithResponses
	// Global gateway (JWT authenticator) clients.
	regionClient *regionv1.ClientWithResponses

	// globalURL is the port-forwarded global gateway, for tests that drive it
	// with raw requests instead of an SDK client (see jwt_test.go).
	globalURL string
)

func TestMain(m *testing.M) {
	restConfig, clientset, err := testenv.SetupK8sClient()
	if err != nil {
		log.Fatalf("Failed to set up k8s client: %v", err)
	}

	regionalPF, err := testenv.StartPortForward(clientset, restConfig, systemNamespace, regionalLabel)
	if err != nil {
		log.Fatalf("Failed to port-forward to regional gateway: %v", err)
	}
	globalPF, err := testenv.StartPortForward(clientset, restConfig, systemNamespace, globalLabel)
	if err != nil {
		regionalPF.Close()
		log.Fatalf("Failed to port-forward to global gateway: %v", err)
	}

	// The two gateways are deployed with different authentication plugins, so a
	// single run exercises both: the regional one verifies dummy tokens, the
	// global one verifies standard signed JWTs. Same admin subject either way —
	// RBAC resolves roles from the subject, not from the token format.
	dummyEditor := authhelper.AdminEditor()
	jwtEditor := authhelper.AdminJWTEditor()

	regionalURL := fmt.Sprintf("http://localhost:%d", regionalPF.LocalPort)
	globalURL = fmt.Sprintf("http://localhost:%d", globalPF.LocalPort)

	if storageClient, err = storagev1.NewClientWithResponses(regionalURL+"/providers/seca.storage", storagev1.WithRequestEditorFn(dummyEditor)); err != nil {
		log.Fatalf("Failed to create storage SDK client: %v", err)
	}
	if workspaceClient, err = workspacev1.NewClientWithResponses(regionalURL+"/providers/seca.workspace", workspacev1.WithRequestEditorFn(dummyEditor)); err != nil {
		log.Fatalf("Failed to create workspace SDK client: %v", err)
	}
	if regionClient, err = regionv1.NewClientWithResponses(globalURL+"/providers/seca.region", regionv1.WithRequestEditorFn(jwtEditor)); err != nil {
		log.Fatalf("Failed to create region SDK client: %v", err)
	}

	log.Println("End-to-end environment ready. Running tests...")
	code := m.Run()

	regionalPF.Close()
	globalPF.Close()
	os.Exit(code)
}
