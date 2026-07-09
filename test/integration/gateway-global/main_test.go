//go:build integration

package integration

import (
	"fmt"
	"log"
	"os"
	"testing"

	authv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"

	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

const (
	testNamespace = "e2e-ecp"
	gatewayLabel  = "app=gateway-global"
)

var (
	regionClient *regionv1.ClientWithResponses
	authClient   *authv1.ClientWithResponses
)

// TestMain wires the suite to the global gateway alone. It exercises the
// gateway's REST↔CR translation for regions, roles and role-assignments and
// never inspects reconciled status, so it needs only the global gateway plus the
// test-data fixtures. Shared k8s-client and port-forward setup lives in testenv.
func TestMain(m *testing.M) {
	restConfig, clientset, err := testenv.SetupK8sClient()
	if err != nil {
		log.Fatalf("Failed to set up k8s client: %v", err)
	}

	pf, err := testenv.StartPortForward(clientset, restConfig, testNamespace, gatewayLabel)
	if err != nil {
		log.Fatalf("Failed to port-forward to global gateway: %v", err)
	}

	baseURL := fmt.Sprintf("http://localhost:%d", pf.LocalPort)
	regionClient, err = regionv1.NewClientWithResponses(baseURL + "/providers/seca.region")
	if err != nil {
		log.Fatalf("Failed to create region SDK client: %v", err)
	}
	authClient, err = authv1.NewClientWithResponses(baseURL + "/providers/seca.authorization")
	if err != nil {
		log.Fatalf("Failed to create authorization SDK client: %v", err)
	}

	log.Println("Test environment ready. Running tests...")
	code := m.Run()
	pf.Close()
	os.Exit(code)
}
