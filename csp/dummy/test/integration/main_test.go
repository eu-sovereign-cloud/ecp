//go:build integration

package integration

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	internetgatewayk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	publicipk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	routetablek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	securitygrouprulek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	securitygroupk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

const (
	testNamespace = "ecp-dummy-delegator"
	pollInterval  = 2 * time.Second
	timeout       = 2 * time.Minute
	// dependencyTimeout allows for resources that must wait on a dependency to
	// become active before they themselves can be reconciled (two sequential
	// simulated operations), so it is larger than the single-operation timeout.
	dependencyTimeout = 5 * time.Minute

	testTenant    = "test-tenant"
	testWorkspace = "test-workspace"
)

var (
	dynamicClient         dynamic.Interface
	testLogger            *slog.Logger
	networkRepo           *k8sadapter.RepoAdapter[*netdom.Network]
	nicRepo               *k8sadapter.RepoAdapter[*nicdom.Nic]
	publicIpRepo          *k8sadapter.RepoAdapter[*publicipdom.PublicIp]
	internetGatewayRepo   *k8sadapter.RepoAdapter[*internetgatewaydom.InternetGateway]
	routeTableRepo        *k8sadapter.RepoAdapter[*routetabledom.RouteTable]
	securityGroupRepo     *k8sadapter.RepoAdapter[*securitygroupdom.SecurityGroup]
	securityGroupRuleRepo *k8sadapter.RepoAdapter[*securitygroupruledom.SecurityGroupRule]
	subnetRepo            *k8sadapter.RepoAdapter[*subnetdom.Subnet]
	instanceRepo          *k8sadapter.RepoAdapter[*instancedom.Instance]
	workspaceRepo         *k8sadapter.RepoAdapter[*wsdom.Workspace]
	blockStorageRepo      *k8sadapter.RepoAdapter[*bsdom.BlockStorage]
	imageRepo             *k8sadapter.RepoAdapter[*imgdom.Image]
	k8sClient             client.Client
)

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		log.Fatalf("Failed to setup integration tests: %v", err)
	}

	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(netk8s.AddToScheme(s))
	utilruntime.Must(nick8s.AddToScheme(s))
	utilruntime.Must(publicipk8s.AddToScheme(s))
	utilruntime.Must(internetgatewayk8s.AddToScheme(s))
	utilruntime.Must(routetablek8s.AddToScheme(s))
	utilruntime.Must(securitygroupk8s.AddToScheme(s))
	utilruntime.Must(securitygrouprulek8s.AddToScheme(s))
	utilruntime.Must(subnetk8s.AddToScheme(s))
	utilruntime.Must(instancek8s.AddToScheme(s))
	utilruntime.Must(wsk8s.AddToScheme(s))
	utilruntime.Must(bsk8s.AddToScheme(s))
	utilruntime.Must(imgk8s.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))

	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := kubeconfig.ClientConfig()
	if err != nil {
		log.Fatalf("Failed to get kubeconfig: %v", err)
	}
	restConfig.QPS = 100
	restConfig.Burst = 200

	k8sClient, err = client.New(restConfig, client.Options{Scheme: s})
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	dynamicClient, err = dynamic.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}

	testLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	networkRepo = k8sadapter.NewRepoAdapter[*netdom.Network](
		dynamicClient,
		netk8s.NetworkGVR,
		testLogger,
		netk8s.Converter,
	)
	nicRepo = k8sadapter.NewRepoAdapter[*nicdom.Nic](
		dynamicClient,
		nick8s.NICGVR,
		testLogger,
		nick8s.Converter,
	)
	publicIpRepo = k8sadapter.NewRepoAdapter[*publicipdom.PublicIp](
		dynamicClient,
		publicipk8s.PublicIPGVR,
		testLogger,
		publicipk8s.Converter,
	)
	internetGatewayRepo = k8sadapter.NewRepoAdapter[*internetgatewaydom.InternetGateway](
		dynamicClient,
		internetgatewayk8s.InternetGatewayGVR,
		testLogger,
		internetgatewayk8s.Converter,
	)
	routeTableRepo = k8sadapter.NewRepoAdapter[*routetabledom.RouteTable](
		dynamicClient,
		routetablek8s.RouteTableGVR,
		testLogger,
		routetablek8s.Converter,
	)
	securityGroupRepo = k8sadapter.NewRepoAdapter[*securitygroupdom.SecurityGroup](
		dynamicClient,
		securitygroupk8s.SecurityGroupGVR,
		testLogger,
		securitygroupk8s.Converter,
	)
	securityGroupRuleRepo = k8sadapter.NewRepoAdapter[*securitygroupruledom.SecurityGroupRule](
		dynamicClient,
		securitygrouprulek8s.SecurityGroupRuleGVR,
		testLogger,
		securitygrouprulek8s.Converter,
	)
	subnetRepo = k8sadapter.NewRepoAdapter[*subnetdom.Subnet](
		dynamicClient,
		subnetk8s.SubnetGVR,
		testLogger,
		subnetk8s.Converter,
	)
	instanceRepo = k8sadapter.NewRepoAdapter[*instancedom.Instance](
		dynamicClient,
		instancek8s.InstanceGVR,
		testLogger,
		instancek8s.Converter,
	)
	blockStorageRepo = k8sadapter.NewRepoAdapter[*bsdom.BlockStorage](
		dynamicClient,
		bsk8s.BlockStorageGVR,
		testLogger,
		bsk8s.Converter,
	)
	workspaceRepo = k8sadapter.NewRepoAdapter[*wsdom.Workspace](
		dynamicClient,
		wsk8s.WorkspaceGVR,
		testLogger,
		wsk8s.Converter,
	)
	imageRepo = k8sadapter.NewRepoAdapter[*imgdom.Image](
		dynamicClient,
		imgk8s.ImageGVR,
		testLogger,
		imgk8s.Converter,
	)
	if err := waitForNamespace(context.Background(), testNamespace); err != nil {
		log.Fatalf("Failed to wait for namespace %s: %v", testNamespace, err)
	}

	if err := createTestNamespaces(context.Background()); err != nil {
		log.Fatalf("Failed to create test namespaces: %v", err)
	}

	code := m.Run()

	if err := teardown(); err != nil {
		log.Printf("Failed to teardown integration tests: %v", err)
	}

	os.Exit(code)
}

func setup() error {
	log.Println("Setting up KIND cluster for integration tests...")
	cmd := exec.Command("make", "-C", "../../", "kind-start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func teardown() error {
	log.Println("Tearing down KIND cluster...")
	cmd := exec.Command("make", "-C", "../../", "kind-stop")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForNamespace(ctx context.Context, namespace string) error {
	log.Printf("Waiting for namespace %s to be created...", namespace)

	return wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		var ns corev1.Namespace
		err := k8sClient.Get(ctx, client.ObjectKey{Name: namespace}, &ns)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func createTestNamespaces(ctx context.Context) error {
	log.Println("Creating test namespaces...")
	networkScopedRouteTable := &routetabledom.RouteTable{
		RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{Network: testNetwork},
	}
	networkScopedRouteTable.Tenant = testTenant
	networkScopedRouteTable.Workspace = testWorkspace

	nsToCreate := []string{
		k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: "test-tenant"}),
		k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"}),
		k8sadapter.ComputeNetworkNamespace(networkScopedRouteTable),
	}

	for _, nsName := range nsToCreate {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
		if err := k8sClient.Create(ctx, ns); err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create namespace %s: %w", nsName, err)
		}
	}

	return nil
}
