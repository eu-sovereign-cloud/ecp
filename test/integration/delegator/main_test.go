//go:build integration

package integration

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	routetablek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"

	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

const (
	pollInterval = 2 * time.Second
	timeout      = 5 * time.Minute
	// teardownTimeout bounds the whole fixture teardown. Deletion is a handful of
	// reconciles with the dummy plugin, so anything longer means a fixture is wedged and
	// waiting out the full timeout only delays the suite's exit code.
	teardownTimeout = 2 * time.Minute
	testTenant      = "test-tenant"
	testWorkspace   = "test-workspace"
	// testNetwork is the parent network that the networked-scope resources (subnet,
	// route-table) live under. TestMain creates it through networkRepo, which provisions
	// its per-network namespace the same way the gateway does; the network resource
	// itself is exercised separately in network_test.go.
	testNetwork = "test-network"
	testRegion  = "ITBG-Bergamo"
	// networkCIDR is the fixture network's address space. The dummy plugin does not
	// validate it; it only has to be a well-formed CIDR.
	networkCIDR = "10.30.0.0/16"
	// sourceBlockStorage is a workspace-scoped block storage that image tests depend
	// on: images reference "block-storages/source-bs" and stay pending until it is
	// active.
	sourceBlockStorage = "source-bs"
)

var (
	dynamicClient    dynamic.Interface
	clientset        kubernetes.Interface
	testLogger       *slog.Logger
	workspaceRepo    persistence.Repo[*wsdom.Workspace]
	blockStorageRepo persistence.Repo[*bsdom.BlockStorage]
	imageRepo        persistence.Repo[*imgdom.Image]
	networkRepo      persistence.Repo[*netdom.Network]
	subnetRepo       persistence.Repo[*subnetdom.Subnet]
	routeTableRepo   persistence.Repo[*routetabledom.RouteTable]
	instanceRepo     persistence.Repo[*instancedom.Instance]
	k8sClient        client.Client
)

func TestMain(m *testing.M) {
	// Initialize k8s scheme for client-go
	s := runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(s))
	utilruntime.Must(wsk8s.AddToScheme(s))
	utilruntime.Must(bsk8s.AddToScheme(s))
	utilruntime.Must(imgk8s.AddToScheme(s))
	utilruntime.Must(netk8s.AddToScheme(s))
	utilruntime.Must(subnetk8s.AddToScheme(s))
	utilruntime.Must(routetablek8s.AddToScheme(s))
	utilruntime.Must(instancek8s.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))

	restConfig, err := testenv.RestConfig()
	if err != nil {
		log.Fatalf("Failed to get kubeconfig: %v", err)
	}
	restConfig.QPS = 100
	restConfig.Burst = 200

	k8sClient, err = client.New(restConfig, client.Options{Scheme: s})
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	// Initialize dynamic clientset
	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	// Initialize dynamic client
	dynamicClient, err = dynamic.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}

	// Initialize test logger
	testLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize repositories
	blockStorageRepo = k8sadapter.NewRepoAdapter(
		dynamicClient,
		bsk8s.BlockStorageGVR,
		testLogger,
		bsk8s.BlockStorageToCR,
		bsk8s.BlockStorageFromCR,
	)

	imageRepo = k8sadapter.NewRepoAdapter(
		dynamicClient,
		imgk8s.ImageGVR,
		testLogger,
		imgk8s.ImageToCR,
		imgk8s.ImageFromCR,
	)

	// Namespace-managing, exactly as the regional gateway wires it: creating a Network
	// provisions the per-network namespace its children live in, and deleting one is
	// refused while that namespace still holds any of ChildResourceGVRs.
	networkRepo = k8sadapter.NewNamespaceManagingRepoAdapter(
		dynamicClient,
		clientset,
		netk8s.NetworkGVR,
		testLogger,
		netk8s.NetworkToCR,
		netk8s.NetworkFromCR,
		k8sadapter.NetworkChildren,
		netk8s.ChildResourceGVRs,
	)

	subnetRepo = k8sadapter.NewRepoAdapter(
		dynamicClient,
		subnetk8s.SubnetGVR,
		testLogger,
		subnetk8s.SubnetToCR,
		subnetk8s.SubnetFromCR,
	)

	routeTableRepo = k8sadapter.NewRepoAdapter(
		dynamicClient,
		routetablek8s.RouteTableGVR,
		testLogger,
		routetablek8s.RouteTableToCR,
		routetablek8s.RouteTableFromCR,
	)

	instanceRepo = k8sadapter.NewRepoAdapter(
		dynamicClient,
		instancek8s.InstanceGVR,
		testLogger,
		instancek8s.InstanceToCR,
		instancek8s.InstanceFromCR,
	)

	workspaceRepo = k8sadapter.NewNamespaceManagingRepoAdapter(
		dynamicClient,
		clientset,
		wsk8s.WorkspaceGVR,
		testLogger,
		wsk8s.WorkspaceToCR,
		wsk8s.WorkspaceFromCR,
		k8sadapter.WorkspaceChildren,
		wsk8s.ChildResourceGVRs,
	)

	// Provide Workspace for BlockStorage tests
	if err := createTestWorkspace(context.Background(), workspaceRepo); err != nil {
		log.Fatalf("Failed to create test workspace: %v", err)
	}

	// Provide the parent network of the networked-scope resources (subnet, route-table).
	// Creating it through networkRepo provisions its per-network namespace, so nothing
	// here hand-creates a namespace the product already owns.
	if err := createTestNetwork(context.Background(), networkRepo); err != nil {
		log.Fatalf("Failed to create test network: %v", err)
	}

	// Provide the source block storage that Image tests depend on. Images reference
	// it with a workspace-qualified "block-storages/source-bs" reference and only
	// become active once it is active, so it must exist and be reconciled before the
	// suite runs. createTestWorkspace above provisions the workspace namespace it
	// lives in.
	if err := createSourceBlockStorage(context.Background(), blockStorageRepo); err != nil {
		log.Fatalf("Failed to create source block storage: %v", err)
	}
	if err := waitForBlockStorageActive(context.Background(), blockStorageRepo, sourceBlockStorage); err != nil {
		log.Fatalf("Source block storage %q did not become active: %v", sourceBlockStorage, err)
	}

	// When running the test suite
	exitCode := m.Run()

	// Cleanup fixtures, innermost first: every delete below is refused while the
	// namespace it owns still holds children, so each one has to be gone — not merely
	// requested — before the next runs. The source block storage additionally cannot go
	// while an image still references it, so image tests must have cleaned up by now.
	teardown(context.Background())

	os.Exit(exitCode)
}

// newTestWorkspace builds the workspace every workspace-scoped fixture and test lives in.
// Creating it through workspaceRepo provisions both the tenant namespace and the
// workspace's own child namespace.
func newTestWorkspace() *wsdom.Workspace {
	return &wsdom.Workspace{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: testWorkspace,
			},
			Scope: resource.Scope{
				Tenant: testTenant,
			},
		},
	}
}

func createTestWorkspace(ctx context.Context, workspaceRepo persistence.Repo[*wsdom.Workspace]) error {
	// Tolerate a leftover fixture from a previously interrupted run (e.g. a test
	// timeout that skipped teardown) so the suite is not permanently wedged.
	if _, err := workspaceRepo.Create(ctx, newTestWorkspace()); err != nil && !errors.Is(err, kernel.ErrAlreadyExists) {
		return err
	}
	return nil
}

func cleanupTestWorkspace(ctx context.Context, workspaceRepo persistence.Repo[*wsdom.Workspace]) error {
	return workspaceRepo.Delete(ctx, newTestWorkspace())
}

// teardown removes the TestMain fixtures in reverse dependency order, waiting for each
// CR to actually disappear before moving outward. Best-effort: a wedged fixture must not
// mask the suite's own exit code, and createX tolerates a leftover on the next run.
func teardown(ctx context.Context) {
	// One shared budget for the three waits: a wedged fixture (a test that left a child
	// behind, so the parent's delete keeps being refused) must not turn teardown into
	// three full polling timeouts back to back.
	ctx, cancel := context.WithTimeout(ctx, teardownTimeout)
	defer cancel()

	_ = cleanupSourceBlockStorage(ctx, blockStorageRepo)
	_ = waitGone(ctx, blockStorageRepo, newSourceBlockStorage())

	_ = cleanupTestNetwork(ctx, networkRepo)
	_ = waitGone(ctx, networkRepo, newTestNetwork())

	_ = cleanupTestWorkspace(ctx, workspaceRepo)
	_ = waitGone(ctx, workspaceRepo, newTestWorkspace())
}

// waitGone polls until obj can no longer be loaded. Deletion is asynchronous — the
// controller holds a finalizer until its plugin and its namespace cleanup are done — and
// a parent's delete is refused (409) while any child CR is still present, so the caller
// has to wait rather than just fire the delete.
func waitGone[T persistence.IdentifiableResource](ctx context.Context, repo persistence.Repo[T], obj T) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		loaded := obj
		if err := repo.Load(ctx, &loaded); err != nil {
			return errors.Is(err, kernel.ErrNotFound), nil
		}
		return false, nil
	})
}

// newTestNetwork builds the parent network of the network-scoped fixtures. Creating it
// through networkRepo provisions testNetworkNamespace().
func newTestNetwork() *netdom.Network {
	return &netdom.Network{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: testNetwork},
			Scope:          resource.Scope{Tenant: testTenant, Workspace: testWorkspace},
		},
		Spec: netdom.NetworkSpec{
			CIDR:   netdom.CIDR{IPv4: networkCIDR},
			SkuRef: commondomain.Reference{Resource: "sku-1"},
		},
	}
}

func createTestNetwork(ctx context.Context, repo persistence.Repo[*netdom.Network]) error {
	// Tolerate a leftover fixture from a previously interrupted run so the suite is not
	// permanently wedged.
	if _, err := repo.Create(ctx, newTestNetwork()); err != nil && !errors.Is(err, kernel.ErrAlreadyExists) {
		return err
	}
	return nil
}

func cleanupTestNetwork(ctx context.Context, repo persistence.Repo[*netdom.Network]) error {
	return repo.Delete(ctx, newTestNetwork())
}

// testNetworkNamespace is the per-network namespace the network-scoped fixtures (subnet,
// route-table) live in: hex(sha3-224(tenant/workspace/network)). Derived from a
// network-scoped identity because that is the shape ComputeNetworkNamespace takes.
func testNetworkNamespace() string {
	rt := &routetabledom.RouteTable{
		RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{Network: testNetwork},
	}
	rt.Tenant = testTenant
	rt.Workspace = testWorkspace
	return k8sadapter.ComputeNetworkNamespace(rt)
}

// newSourceBlockStorage builds the workspace-scoped block storage that image tests
// depend on. It lives in the test workspace; image resources reference it with a
// workspace-qualified "block-storages/source-bs" reference.
func newSourceBlockStorage() *bsdom.BlockStorage {
	return &bsdom.BlockStorage{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: sourceBlockStorage,
			},
			Scope: resource.Scope{
				Tenant:    testTenant,
				Workspace: testWorkspace,
			},
		},
		Spec: bsdom.BlockStorageSpec{
			SizeGB: 1,
			SkuRef: commondomain.Reference{
				Region:   testRegion,
				Resource: "sku-1",
			},
		},
	}
}

func createSourceBlockStorage(ctx context.Context, repo persistence.Repo[*bsdom.BlockStorage]) error {
	// Tolerate a leftover fixture from a previously interrupted run so the suite is
	// not permanently wedged; waitForBlockStorageActive still ensures it is ready.
	if _, err := repo.Create(ctx, newSourceBlockStorage()); err != nil && !errors.Is(err, kernel.ErrAlreadyExists) {
		return err
	}
	return nil
}

func cleanupSourceBlockStorage(ctx context.Context, repo persistence.Repo[*bsdom.BlockStorage]) error {
	return repo.Delete(ctx, newSourceBlockStorage())
}

// waitForBlockStorageActive polls until the named workspace-scoped block storage
// reports an active state.
func waitForBlockStorageActive(ctx context.Context, repo persistence.Repo[*bsdom.BlockStorage], name string) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		loaded := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: name,
				},
				Scope: resource.Scope{
					Tenant:    testTenant,
					Workspace: testWorkspace,
				},
			},
		}
		if err := repo.Load(ctx, &loaded); err != nil {
			return false, nil
		}
		return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
	})
}
