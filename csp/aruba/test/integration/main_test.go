//go:build integration

// Package integration is the aruba plugin's own integration suite. Like csp/dummy/test/integration,
// it drives the plugin directly - it creates the SECA CRs and asserts the delegator reconciles them,
// with NO gateway in the path. Unlike the dummy suite it does not stand up its own cluster: the aruba
// backend is the third-party arubacloud-resource-operator, which (plus Aruba credentials) must be
// installed out of band. So the suite connects to the current kube-context and expects the
// delegator-aruba + operator already deployed - the two-phase flow in test/conformance/aruba.
//
//	make test-integration ARUBA_TENANT=ARU-348095   # against a cluster with the operator + creds
//
// A real Aruba account is required: the plugin writes arubacloud.com CRs the operator provisions for
// real, so the resources only reach Active against a genuine tenant. ARUBA_TENANT must name one.
package integration

import (
	"context"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	kres "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	instdom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	instk8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	igwdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	igwk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	pipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	pipk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	rtdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	rtk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	sgdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	sgrdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	sgrk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	sgk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
	subdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	subk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

const (
	// Names are >= 4 characters: Aruba CMP rejects shorter resource names (400 validation).
	workspace = "ecp-int-ws"
	network   = "ecp-int-net"
	region    = "ITBG-Bergamo"

	pollInterval = 3 * time.Second
	// Aruba provisions real cloud resources, so give the backend room; a no-op handler
	// (route-table, security-group, nic) goes Active almost immediately.
	activeTimeout = 4 * time.Minute
	noopTimeout   = 45 * time.Second
	// A real VM boots an OS from its image, materialises its security group and clones a bootable
	// disk - measurably slower than the other resources, so it gets a longer deadline.
	vmActiveTimeout = 10 * time.Minute
	// Editing a live SECA resource triggers a reconcile straight away, and the plugin's update only
	// has to write the arubacloud.com CR - no provisioning - so it lands in seconds.
	updateTimeout = 60 * time.Second

	// Resource names shared across the suite. All >= 4 characters (Aruba rejects shorter names).
	bootName = "boot"
	// imgSrcName is the source volume the boot image is captured from; bootImage is a SECA image
	// name skumap knows (-> Aruba template DE12-001), so the boot disk boots Debian 12.
	imgSrcName = "imgsrc"
	bootImage  = "debian-12"
	igwName    = "igw1"
	rtName     = "rtbl"
	subnetName = "subneta"
	pipName    = "pubip"
	sgName     = "websg"
	sgrName    = "rule1"
	nicName    = "nic1"
	instName   = "vm-1"

	// A well-formed ed25519 public key (format is what Aruba validates); override with ARUBA_SSH_KEY.
	defaultSSHKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJP0fH75tLZbeyBxqtzQiSQFjG4EMJ95eq7QDXOlp68C ecp-int"
)

// tenant must be a real Aruba account for resources to provision; override with ARUBA_TENANT.
var tenant = envOr("ARUBA_TENANT", "test-tenant")

var (
	ctx    = context.Background()
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	dyn dynamic.Interface
	cs  kubernetes.Interface

	tenantNS, wsNS, netNS string

	wsRepo   *k8sadapter.RepoAdapter[*wsdom.Workspace]
	bsRepo   *k8sadapter.RepoAdapter[*bsdom.BlockStorage]
	imgRepo  *k8sadapter.RepoAdapter[*imgdom.Image]
	igwRepo  *k8sadapter.RepoAdapter[*igwdom.InternetGateway]
	rtRepo   *k8sadapter.RepoAdapter[*rtdom.RouteTable]
	netRepo  *k8sadapter.RepoAdapter[*netdom.Network]
	subRepo  *k8sadapter.RepoAdapter[*subdom.Subnet]
	pipRepo  *k8sadapter.RepoAdapter[*pipdom.PublicIp]
	sgRepo   *k8sadapter.RepoAdapter[*sgdom.SecurityGroup]
	sgrRepo  *k8sadapter.RepoAdapter[*sgrdom.SecurityGroupRule]
	nicRepo  *k8sadapter.RepoAdapter[*nicdom.Nic]
	instRepo *k8sadapter.RepoAdapter[*instdom.Instance]
)

func TestMain(m *testing.M) {
	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		log.Fatalf("kubeconfig: %v", err)
	}
	restConfig.QPS, restConfig.Burst = 100, 200
	if dyn, err = dynamic.NewForConfig(restConfig); err != nil {
		log.Fatalf("dynamic client: %v", err)
	}
	if cs, err = kubernetes.NewForConfig(restConfig); err != nil {
		log.Fatalf("clientset: %v", err)
	}

	wsRepo = repo[*wsdom.Workspace](wsk8s.WorkspaceGVR, wsk8s.WorkspaceToCR, wsk8s.WorkspaceFromCR)
	bsRepo = repo[*bsdom.BlockStorage](bsk8s.BlockStorageGVR, bsk8s.BlockStorageToCR, bsk8s.BlockStorageFromCR)
	imgRepo = repo[*imgdom.Image](imgk8s.ImageGVR, imgk8s.ImageToCR, imgk8s.ImageFromCR)
	igwRepo = repo[*igwdom.InternetGateway](igwk8s.InternetGatewayGVR, igwk8s.InternetGatewayToCR, igwk8s.InternetGatewayFromCR)
	rtRepo = repo[*rtdom.RouteTable](rtk8s.RouteTableGVR, rtk8s.RouteTableToCR, rtk8s.RouteTableFromCR)
	netRepo = repo[*netdom.Network](netk8s.NetworkGVR, netk8s.NetworkToCR, netk8s.NetworkFromCR)
	subRepo = repo[*subdom.Subnet](subk8s.SubnetGVR, subk8s.SubnetToCR, subk8s.SubnetFromCR)
	pipRepo = repo[*pipdom.PublicIp](pipk8s.PublicIPGVR, pipk8s.PublicIpToCR, pipk8s.PublicIpFromCR)
	sgRepo = repo[*sgdom.SecurityGroup](sgk8s.SecurityGroupGVR, sgk8s.SecurityGroupToCR, sgk8s.SecurityGroupFromCR)
	sgrRepo = repo[*sgrdom.SecurityGroupRule](sgrk8s.SecurityGroupRuleGVR, sgrk8s.SecurityGroupRuleToCR, sgrk8s.SecurityGroupRuleFromCR)
	nicRepo = repo[*nicdom.Nic](nick8s.NICGVR, nick8s.NicToCR, nick8s.NicFromCR)
	instRepo = repo[*instdom.Instance](instk8s.InstanceGVR, instk8s.InstanceToCR, instk8s.InstanceFromCR)

	// The gateway normally provisions the tenant/workspace/network namespaces; this suite has no
	// gateway, so create them (mirrors csp/dummy/test/integration TestMain).
	tenantNS = k8sadapter.ComputeNamespace(&kres.Scope{Tenant: tenant})
	wsNS = k8sadapter.ComputeNamespace(&kres.Scope{Tenant: tenant, Workspace: workspace})
	netNS = k8sadapter.ComputeNetworkNamespace(&rtdom.RouteTable{RegionalNetworkMetadata: rnMeta("probe")})
	for _, ns := range []string{tenantNS, wsNS, netNS} {
		if err := ensureNamespace(ns); err != nil {
			log.Fatalf("namespace %s: %v", ns, err)
		}
	}

	os.Exit(m.Run())
}

func arubaGVR(resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "arubacloud.com", Version: "v1alpha1", Resource: resource}
}

// arubaPhase returns the phase of the arubacloud.com CR the plugin materialised, or a sentinel
// ("NOTFOUND"/"ERR:...") - used to assert the plugin's output independently of provisioning.
func arubaPhase(resource, name, ns string) string {
	u, err := dyn.Resource(arubaGVR(resource)).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return "NOTFOUND"
		}
		return "ERR:" + err.Error()
	}
	phase, _, _ := unstructured.NestedString(u.Object, "status", "phase")
	return phase
}

// arubaSpec returns the spec of the arubacloud.com CR the plugin materialised, or nil when it
// cannot be read. The update assertions poll, so a missing CR just means "not yet".
func arubaSpec(resource, name, ns string) map[string]any {
	u, err := dyn.Resource(arubaGVR(resource)).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	return u.Object
}

func arubaTags(resource, name, ns string) []string {
	tags, _, _ := unstructured.NestedStringSlice(arubaSpec(resource, name, ns), "spec", "tags")
	return tags
}

func arubaDescription(resource, name, ns string) string {
	description, _, _ := unstructured.NestedString(arubaSpec(resource, name, ns), "spec", "description")
	return description
}

func repo[T persistence.IdentifiableResource](gvr schema.GroupVersionResource,
	toCR k8sadapter.DomainToK8s[T], fromCR k8sadapter.K8sToDomain[T]) *k8sadapter.RepoAdapter[T] {
	return k8sadapter.NewRepoAdapter[T](dyn, gvr, logger, toCR, fromCR)
}

func ensureNamespace(ns string) error {
	_, err := cs.CoreV1().Namespaces().Create(ctx,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// waitGone polls until load reports the resource is absent (NotFound).
func waitGone(load func() error) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, activeTimeout, true, func(context.Context) (bool, error) {
		if err := load(); err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
}
