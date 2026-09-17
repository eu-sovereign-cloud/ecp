package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	sdkcomputeapi "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdknetworkapi "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkstorageapi "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkworkspaceapi "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/frontend/middleware"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/auth"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/httpserver"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/logger"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/metrics"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	rak8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment/backend/kubernetes"
	rolek8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role/backend/kubernetes"
	computerest "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/frontend/rest"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	computeskudom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku"
	computeskuk8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku/backend/kubernetes"
	netrest "github.com/eu-sovereign-cloud/ecp/resource/network/v1/frontend/rest"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	internetgatewayk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	netskudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
	netskuk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku/backend/kubernetes"
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
	storagerest "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/frontend/rest"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku"
	skuk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
	wsrest "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/frontend/rest"
)

var (
	region             string
	regions            []string
	regionalHost       string
	regionalPort       string
	regionalKubeconfig string

	regionalAuthFlags auth.Flags
	regionalKubeFlags kubeclient.ClientFlags
)

var regionalApiServerCMD = &cobra.Command{
	Use:     "regionalapiserver",
	Aliases: []string{"regional"},
	Short:   "The command starts the regional server for the ECP application",
	Long:    `The command starts the regional server for the ECP application`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := logger.New(os.Getenv("APP_ENV"))
		if err := startRegional(logger, regionalHost+":"+regionalPort, regionalKubeconfig); err != nil {
			logger.Error("regional API server failed", slog.Any("error", err))
			os.Exit(1)
		}
	},
}

func init() {
	regionalApiServerCMD.Flags().StringVar(
		&region, "region", "", "The region served by the regional gateway, and the one a request that names none is served as",
	)
	regionalApiServerCMD.Flags().StringSliceVar(
		&regions, "regions", nil,
		"Comma-separated regions this gateway serves; a request selects one with a /regions/<region> path prefix, "+
			"and --region (the first of these when unset) is the default for requests that name none",
	)
	regionalApiServerCMD.Flags().StringVar(
		&regionalHost, "regionalHost", "0.0.0.0", "Host to bind the server to",
	)
	regionalApiServerCMD.Flags().StringVarP(
		&regionalPort, "regionalPort", "p", "8080", "Port to bind the server to",
	)
	regionalApiServerCMD.Flags().StringVar(
		&regionalKubeconfig, "kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"),
		"Path to regional kubeconfig",
	)
	auth.RegisterFlags(regionalApiServerCMD, &regionalAuthFlags)
	kubeclient.RegisterClientFlags(regionalApiServerCMD, &regionalKubeFlags)
	rootCmd.AddCommand(regionalApiServerCMD)
}

// startRegional starts the backend HTTP server on the given address.
func startRegional(logger *slog.Logger, addr string, kubeconfigPath string) error {
	if region == "" {
		region = os.Getenv("REGION")
	}
	if len(regions) == 0 && os.Getenv("REGIONS") != "" {
		regions = strings.Split(os.Getenv("REGIONS"), ",")
	}
	servedRegions, defaultRegion, err := resolveRegions(region, regions)
	if err != nil {
		return err
	}

	logger.Info("Starting regional API server",
		slog.Any("regions", servedRegions), slog.String("default_region", defaultRegion), slog.Any("addr", addr))
	metrics.RegisterUpstreamObserver()

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Warn(
			"could not get in-cluster config, falling back to kubeconfig file",
			slog.Any("error", err),
		)
		inClusterConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return fmt.Errorf("build kubeconfig %s: %w", kubeconfigPath, err)
		}
	}

	if err := regionalKubeFlags.ApplyToConfig(inClusterConfig); err != nil {
		return fmt.Errorf("apply kube client flags: %w", err)
	}
	logger.Info("kube client rate limit",
		slog.Float64("kube_qps", float64(regionalKubeFlags.QPS)),
		slog.Int("kube_burst", regionalKubeFlags.Burst),
	)

	client, err := kubeclient.NewFromConfig(inClusterConfig)
	if err != nil {
		return fmt.Errorf("create kubeclient: %w", err)
	}

	// Create a shared mux for all regional handlers
	mux := http.NewServeMux()
	readiness := httpserver.NewReadiness()
	// Probes are unauthenticated and registered before provider routes so kubelet
	// can hit them while the process is still wiring (readyz stays 503 until Set).
	httpserver.RegisterProbes(mux, readiness, client.CheckAPIServer)

	// Compute adapters
	instanceReaderAdapter := k8sadapter.NewReaderAdapter[*instancedom.Instance](
		client.Client,
		instancek8s.InstanceGVR,
		logger,
		instancek8s.InstanceFromCR,
	).RegionScoped()
	instanceWriterAdapter := k8sadapter.NewWriterAdapter[*instancedom.Instance](
		client.Client,
		instancek8s.InstanceGVR,
		logger,
		instancek8s.Converter,
	)
	instanceSKUReaderAdapter := k8sadapter.NewReaderAdapter[*computeskudom.InstanceSKU](
		client.Client,
		computeskuk8s.InstanceSKUGVR,
		logger,
		computeskuk8s.InstanceSKUFromCR,
	).RegionScoped()
	// Metrics endpoint — unauthenticated, mounted outside provider HandlerWithOptions.
	mux.Handle("/metrics", metrics.Handler())

	// RBAC reader adapters used by the authorization checker. Not RegionScoped: Role and
	// RoleAssignment are tenant-scoped and carry no region label, so filtering on one would
	// resolve every caller to no roles at all.
	roleReaderAdapter := k8sadapter.NewReaderAdapter[*roledom.Role](
		client.Client,
		rolek8s.RoleGVR,
		logger,
		rolek8s.RoleFromCR,
	)
	roleAssignmentReaderAdapter := k8sadapter.NewReaderAdapter[*radom.RoleAssignment](
		client.Client,
		rak8s.RoleAssignmentGVR,
		logger,
		rak8s.RoleAssignmentFromCR,
	)

	// Build the authenticator and RBAC checker (both nil when --auth-enabled is not set).
	authenticator, checker, err := auth.Build(&regionalAuthFlags, client.Client, roleReaderAdapter, roleAssignmentReaderAdapter, logger)
	if err != nil {
		return fmt.Errorf("build auth chain: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start the informer-backed checker when --authz-cache is enabled.
	// Same ctx as Serve so SIGTERM cancels informers during drain.
	if err := auth.StartChecker(ctx, checker, logger); err != nil {
		return fmt.Errorf("start authz cache: %w", err)
	}

	sdkcomputeapi.HandlerWithOptions(
		&computerest.Handler{
			InstanceReader: instanceReaderAdapter,
			InstanceWriter: instanceWriterAdapter,
			SKUReader:      instanceSKUReaderAdapter,
			Logger:         logger,
			Region:         defaultRegion,
		},
		sdkcomputeapi.StdHTTPServerOptions{
			BaseURL:    "/providers/seca.compute",
			BaseRouter: mux,
			Middlewares: auth.ProviderMWs[sdkcomputeapi.MiddlewareFunc](&regionalAuthFlags, authenticator, checker, "seca.compute",
				"/providers/seca.compute", defaultRegion, logger),
			ErrorHandlerFunc: nil,
		},
	)

	// Network adapters
	netReaderAdapter := k8sadapter.NewReaderAdapter[*netdom.Network](
		client.Client,
		netk8s.NetworkGVR,
		logger,
		netk8s.NetworkFromCR,
	).RegionScoped()
	netWriterAdapter := k8sadapter.NewNamespaceManagingWriterAdapter[*netdom.Network](
		client.Client,
		client.ClientSet,
		netk8s.NetworkGVR,
		logger,
		netk8s.Converter,
		k8sadapter.NetworkChildren,
		netk8s.ChildResourceGVRs,
	)
	netSKUReaderAdapter := k8sadapter.NewReaderAdapter[*netskudom.NetworkSKU](
		client.Client,
		netskuk8s.NetworkSKUGVR,
		logger,
		netskuk8s.NetworkSKUFromCR,
	).RegionScoped()
	nicReaderAdapter := k8sadapter.NewReaderAdapter[*nicdom.Nic](
		client.Client,
		nick8s.NICGVR,
		logger,
		nick8s.NicFromCR,
	).RegionScoped()
	nicWriterAdapter := k8sadapter.NewWriterAdapter[*nicdom.Nic](
		client.Client,
		nick8s.NICGVR,
		logger,
		nick8s.Converter,
	)
	publicIpReaderAdapter := k8sadapter.NewReaderAdapter[*publicipdom.PublicIp](
		client.Client,
		publicipk8s.PublicIPGVR,
		logger,
		publicipk8s.PublicIpFromCR,
	).RegionScoped()
	publicIpWriterAdapter := k8sadapter.NewWriterAdapter[*publicipdom.PublicIp](
		client.Client,
		publicipk8s.PublicIPGVR,
		logger,
		publicipk8s.Converter,
	)
	internetGatewayReaderAdapter := k8sadapter.NewReaderAdapter[*internetgatewaydom.InternetGateway](
		client.Client,
		internetgatewayk8s.InternetGatewayGVR,
		logger,
		internetgatewayk8s.InternetGatewayFromCR,
	).RegionScoped()
	internetGatewayWriterAdapter := k8sadapter.NewWriterAdapter[*internetgatewaydom.InternetGateway](
		client.Client,
		internetgatewayk8s.InternetGatewayGVR,
		logger,
		internetgatewayk8s.Converter,
	)
	routeTableReaderAdapter := k8sadapter.NewReaderAdapter[*routetabledom.RouteTable](
		client.Client,
		routetablek8s.RouteTableGVR,
		logger,
		routetablek8s.RouteTableFromCR,
	).RegionScoped()
	routeTableWriterAdapter := k8sadapter.NewWriterAdapter[*routetabledom.RouteTable](
		client.Client,
		routetablek8s.RouteTableGVR,
		logger,
		routetablek8s.Converter,
	)
	subnetReaderAdapter := k8sadapter.NewReaderAdapter[*subnetdom.Subnet](
		client.Client,
		subnetk8s.SubnetGVR,
		logger,
		subnetk8s.SubnetFromCR,
	).RegionScoped()
	subnetWriterAdapter := k8sadapter.NewWriterAdapter[*subnetdom.Subnet](
		client.Client,
		subnetk8s.SubnetGVR,
		logger,
		subnetk8s.Converter,
	)
	securityGroupReaderAdapter := k8sadapter.NewReaderAdapter[*securitygroupdom.SecurityGroup](
		client.Client,
		securitygroupk8s.SecurityGroupGVR,
		logger,
		securitygroupk8s.SecurityGroupFromCR,
	).RegionScoped()
	securityGroupWriterAdapter := k8sadapter.NewWriterAdapter[*securitygroupdom.SecurityGroup](
		client.Client,
		securitygroupk8s.SecurityGroupGVR,
		logger,
		securitygroupk8s.Converter,
	)
	securityGroupRuleReaderAdapter := k8sadapter.NewReaderAdapter[*securitygroupruledom.SecurityGroupRule](
		client.Client,
		securitygrouprulek8s.SecurityGroupRuleGVR,
		logger,
		securitygrouprulek8s.SecurityGroupRuleFromCR,
	).RegionScoped()
	securityGroupRuleWriterAdapter := k8sadapter.NewWriterAdapter[*securitygroupruledom.SecurityGroupRule](
		client.Client,
		securitygrouprulek8s.SecurityGroupRuleGVR,
		logger,
		securitygrouprulek8s.Converter,
	)

	sdknetworkapi.HandlerWithOptions(
		&netrest.Handler{
			NetworkReader:           netReaderAdapter,
			NetworkWriter:           netWriterAdapter,
			SKUReader:               netSKUReaderAdapter,
			NicReader:               nicReaderAdapter,
			NicWriter:               nicWriterAdapter,
			PublicIpReader:          publicIpReaderAdapter,
			PublicIpWriter:          publicIpWriterAdapter,
			InternetGatewayReader:   internetGatewayReaderAdapter,
			InternetGatewayWriter:   internetGatewayWriterAdapter,
			RouteTableReader:        routeTableReaderAdapter,
			RouteTableWriter:        routeTableWriterAdapter,
			SubnetReader:            subnetReaderAdapter,
			SubnetWriter:            subnetWriterAdapter,
			SecurityGroupReader:     securityGroupReaderAdapter,
			SecurityGroupWriter:     securityGroupWriterAdapter,
			SecurityGroupRuleReader: securityGroupRuleReaderAdapter,
			SecurityGroupRuleWriter: securityGroupRuleWriterAdapter,
			Logger:                  logger,
			Region:                  defaultRegion,
		},
		sdknetworkapi.StdHTTPServerOptions{
			BaseURL:          "/providers/seca.network",
			BaseRouter:       mux,
			Middlewares:      auth.ProviderMWs[sdknetworkapi.MiddlewareFunc](&regionalAuthFlags, authenticator, checker, "seca.network", "/providers/seca.network", defaultRegion, logger),
			ErrorHandlerFunc: nil,
		},
	)

	// Storage adapters
	bsReaderAdapter := k8sadapter.NewReaderAdapter[*bsdom.BlockStorage](
		client.Client,
		bsk8s.BlockStorageGVR,
		logger,
		bsk8s.BlockStorageFromCR,
	).RegionScoped()
	bsWriterAdapter := k8sadapter.NewWriterAdapter[*bsdom.BlockStorage](
		client.Client,
		bsk8s.BlockStorageGVR,
		logger,
		bsk8s.Converter,
	)
	skuReaderAdapter := k8sadapter.NewReaderAdapter[*skudom.StorageSKU](
		client.Client,
		skuk8s.StorageSKUGVR,
		logger,
		skuk8s.StorageSKUFromCR,
	).RegionScoped()
	imgReaderAdapter := k8sadapter.NewReaderAdapter[*imgdom.Image](
		client.Client,
		imgk8s.ImageGVR,
		logger,
		imgk8s.ImageFromCR,
	).RegionScoped()
	imgWriterAdapter := k8sadapter.NewWriterAdapter[*imgdom.Image](
		client.Client,
		imgk8s.ImageGVR,
		logger,
		imgk8s.Converter,
	)

	sdkstorageapi.HandlerWithOptions(
		&storagerest.Handler{
			BlockStorageReader: bsReaderAdapter,
			BlockStorageWriter: bsWriterAdapter,
			ImageReader:        imgReaderAdapter,
			ImageWriter:        imgWriterAdapter,
			SKUReader:          skuReaderAdapter,
			Logger:             logger,
			Region:             defaultRegion,
		},
		sdkstorageapi.StdHTTPServerOptions{
			BaseURL:    "/providers/seca.storage",
			BaseRouter: mux,
			Middlewares: auth.ProviderMWs[sdkstorageapi.MiddlewareFunc](&regionalAuthFlags, authenticator, checker, "seca.storage",
				"/providers/seca.storage", defaultRegion, logger),
			ErrorHandlerFunc: nil,
		},
	)

	// Workspace adapters
	wsWriterAdapter := k8sadapter.NewNamespaceManagingWriterAdapter[*wsdom.Workspace](
		client.Client,
		client.ClientSet,
		wsk8s.WorkspaceGVR,
		logger,
		wsk8s.Converter,
		k8sadapter.WorkspaceChildren,
		wsk8s.ChildResourceGVRs,
	)
	wsReaderAdapter := k8sadapter.NewReaderAdapter[*wsdom.Workspace](
		client.Client,
		wsk8s.WorkspaceGVR,
		logger,
		wsk8s.WorkspaceFromCR,
	).RegionScoped()

	sdkworkspaceapi.HandlerWithOptions(
		&wsrest.Handler{
			Reader: wsReaderAdapter,
			Writer: wsWriterAdapter,
			Logger: logger,
			Region: defaultRegion,
		},
		sdkworkspaceapi.StdHTTPServerOptions{
			BaseURL:    "/providers/seca.workspace",
			BaseRouter: mux,
			Middlewares: auth.ProviderMWs[sdkworkspaceapi.MiddlewareFunc](&regionalAuthFlags, authenticator, checker, "seca.workspace",
				"/providers/seca.workspace", defaultRegion, logger),
			ErrorHandlerFunc: nil,
		},
	)

	// Outermost: it resolves the region and strips the /regions/<region> prefix before the
	// mux matches, so every provider route — and the metrics route label — is the same
	// whether or not the caller named a region.
	regionRouter := middleware.NewRegionRouter(servedRegions, defaultRegion, logger)
	httpServer := httpserver.New(
		httpserver.Options{
			Addr:    addr,
			Handler: regionRouter(metrics.HTTPMiddleware(mux)),
			Logger:  logger,
		},
	)
	// Open the readiness gate only after full wiring; Serve clears it on SIGTERM.
	readiness.Set(true)
	logger.Info("Regional API server started successfully")
	if err := httpserver.Serve(ctx, httpServer, logger, readiness); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	logger.Info("Regional API server shut down gracefully")
	return nil
}
