//go:build integration

package integration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	authhelper "github.com/eu-sovereign-cloud/ecp/test/internal/authhelper"

	computev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	networkv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	workspacev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

const (
	systemNamespace = "e2e-ecp"
	gatewayLabel    = "app=gateway-regional"
	testTenant      = "test-tenant"
	testWorkspace   = "test-workspace"
	// sourceBlockStorage is a workspace-scoped block storage that image tests
	// reference via "block-storages/source-bs". This suite does not wait for it to
	// reconcile — it only has to exist so the image reference resolves.
	sourceBlockStorage = "source-bs"
)

var (
	computeClient   *computev1.ClientWithResponses
	networkClient   *networkv1.ClientWithResponses
	storageClient   *storagev1.ClientWithResponses
	workspaceClient *workspacev1.ClientWithResponses

	regionalLocalPort uint16
)

// TestMain wires the suite to the regional gateway alone. The regional suite
// exercises the gateway's REST↔CR translation (create, read-back, update, delete)
// and never inspects reconciled status. Reconciliation to Active is the delegator's
// job and is covered by the delegator suite, so this suite needs neither the
// delegator nor the global gateway — only the regional gateway and the test-data
// fixtures (tenant namespace + storage SKUs). Shared k8s-client and port-forward
// setup lives in testenv.
func TestMain(m *testing.M) {
	restConfig, clientset, err := testenv.SetupK8sClient()
	if err != nil {
		log.Fatalf("Failed to set up k8s client: %v", err)
	}

	pf, err := testenv.StartPortForward(clientset, restConfig, systemNamespace, gatewayLabel)
	if err != nil {
		log.Fatalf("Regional gateway port-forward failed: %v", err)
	}
	regionalLocalPort = pf.LocalPort

	editor := authhelper.AdminEditor()

	// SECA SDK clients, all pointing at the regional gateway.
	baseURL := fmt.Sprintf("http://localhost:%d", pf.LocalPort)
	workspaceClient, err = workspacev1.NewClientWithResponses(baseURL+"/providers/seca.workspace", workspacev1.WithRequestEditorFn(editor))
	if err != nil {
		log.Fatalf("Failed to create workspace SDK client: %v", err)
	}
	storageClient, err = storagev1.NewClientWithResponses(baseURL+"/providers/seca.storage", storagev1.WithRequestEditorFn(editor))
	if err != nil {
		log.Fatalf("Failed to create storage SDK client: %v", err)
	}
	networkClient, err = networkv1.NewClientWithResponses(baseURL+"/providers/seca.network", networkv1.WithRequestEditorFn(editor))
	if err != nil {
		log.Fatalf("Failed to create network SDK client: %v", err)
	}
	computeClient, err = computev1.NewClientWithResponses(baseURL+"/providers/seca.compute", computev1.WithRequestEditorFn(editor))
	if err != nil {
		log.Fatalf("Failed to create compute SDK client: %v", err)
	}

	// Provision shared fixtures through the gateway itself. Creating the workspace
	// synchronously provisions its namespace (the gateway uses a namespace-managing
	// adapter), which the block storage tests create into; the source block storage
	// is what the image tests reference.
	if err := ensureTestWorkspace(context.Background()); err != nil {
		log.Fatalf("Failed to create test workspace: %v", err)
	}
	if err := ensureSourceBlockStorage(context.Background()); err != nil {
		log.Fatalf("Failed to create source block storage: %v", err)
	}

	exitCode := m.Run()

	// Best-effort cleanup of the shared fixtures.
	_, _ = storageClient.DeleteBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, sourceBlockStorage, nil)
	_, _ = workspaceClient.DeleteWorkspaceWithResponse(context.Background(), testTenant, testWorkspace, nil)

	pf.Close()
	os.Exit(exitCode)
}

// ensureTestWorkspace creates the shared test workspace through the gateway. The
// create is an upsert, so it tolerates a leftover fixture from a previous run.
func ensureTestWorkspace(ctx context.Context) error {
	resp, err := workspaceClient.CreateOrUpdateWorkspaceWithResponse(ctx, testTenant, testWorkspace, nil, schema.Workspace{})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status %d creating workspace %q", resp.StatusCode(), testWorkspace)
	}
	return nil
}

// ensureSourceBlockStorage creates the workspace-scoped source block storage that
// image tests reference. The create is an upsert, so it tolerates a leftover fixture.
func ensureSourceBlockStorage(ctx context.Context) error {
	body := schema.BlockStorage{
		Spec: schema.BlockStorageSpec{
			SizeGB: 1,
			SkuRef: schema.Reference{Resource: "sku-1"},
		},
	}
	resp, err := storageClient.CreateOrUpdateBlockStorageWithResponse(ctx, testTenant, testWorkspace, sourceBlockStorage, nil, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status %d creating source block storage %q", resp.StatusCode(), sourceBlockStorage)
	}
	return nil
}
