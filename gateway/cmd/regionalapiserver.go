package cmd

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	sdkcomputeapi "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdknetworkapi "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkstorageapi "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkworkspaceapi "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
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
	netrest "github.com/eu-sovereign-cloud/ecp/resource/network/v1/frontend/rest"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	netskudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
	netskuk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku/backend/kubernetes"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
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
	regionalHost       string
	regionalPort       string
	regionalKubeconfig string

	regionalAuthFlags auth.Flags
)

var regionalApiServerCMD = &cobra.Command{
	Use:     "regionalapiserver",
	Aliases: []string{"regional"},
	Short:   "The command starts the regional server for the ECP application",
	Long:    `The command starts the regional server for the ECP application`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := logger.New(os.Getenv("APP_ENV"))
		startRegional(logger, regionalHost+":"+regionalPort, regionalKubeconfig)
	},
}

func init() {
	regionalApiServerCMD.Flags().StringVar(
		&region, "region", "", "The region served by the regional gateway",
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
	rootCmd.AddCommand(regionalApiServerCMD)
}

// startRegional starts the backend HTTP server on the given address.
func startRegional(logger *slog.Logger, addr string, kubeconfigPath string) {
	if region == "" {
		region = os.Getenv("REGION")
	}
	config.Singleton().SetRegion(region)

	logger.Info("Starting regional API server", slog.String("region", config.Singleton().Region()), slog.Any("addr", addr))

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Warn(
			"could not get in-cluster config, falling back to kubeconfig file",
			slog.Any("error", err),
		)
		inClusterConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			logger.Error(
				"failed to build kubeconfig", "path", kubeconfigPath, slog.Any("error", err),
			)
			log.Fatal(err, " - failed to build kubeconfig")
		}
	}

	client, err := kubeclient.NewFromConfig(inClusterConfig)
	if err != nil {
		logger.Error("failed to create kubeclient", slog.Any("error", err))
		log.Fatal(err, " - failed to create kubeclient")
	}

	// Create a shared mux for all regional handlers.
	mux := http.NewServeMux()

	// Metrics endpoint — unauthenticated, mounted outside provider HandlerWithOptions.
	mux.Handle("/metrics", metrics.Handler())

	// RBAC reader adapters used by the authorization checker.
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
		logger.Error("failed to build auth chain", slog.Any("error", err))
		log.Fatal(err, " - failed to build auth chain")
	}

	// Start the informer-backed checker when --authz-cache is enabled.
	if err := auth.StartChecker(context.Background(), checker, logger); err != nil {
		logger.Error("failed to start authz cache", slog.Any("error", err))
		log.Fatal(err, " - failed to start authz cache")
	}

	// Compute (stub — not yet implemented)
	sdkcomputeapi.HandlerWithOptions(
		&computerest.Handler{Logger: logger},
		sdkcomputeapi.StdHTTPServerOptions{
			BaseURL:          "/providers/seca.compute",
			BaseRouter:       mux,
			Middlewares:      auth.ProviderMWs[sdkcomputeapi.MiddlewareFunc](authenticator, checker, "seca.compute", "/providers/seca.compute", logger),
			ErrorHandlerFunc: nil,
		},
	)

	// Network adapters
	netReaderAdapter := k8sadapter.NewReaderAdapter[*netdom.Network](
		client.Client,
		netk8s.NetworkGVR,
		logger,
		netk8s.NetworkFromCR,
	)
	netWriterAdapter := k8sadapter.NewWriterAdapter[*netdom.Network](
		client.Client,
		netk8s.NetworkGVR,
		logger,
		netk8s.NetworkToCR,
		netk8s.NetworkFromCR,
	)
	netSKUReaderAdapter := k8sadapter.NewReaderAdapter[*netskudom.NetworkSKU](
		client.Client,
		netskuk8s.NetworkSKUGVR,
		logger,
		netskuk8s.NetworkSKUFromCR,
	)
	nicReaderAdapter := k8sadapter.NewReaderAdapter[*nicdom.Nic](
		client.Client,
		nick8s.NICGVR,
		logger,
		nick8s.NicFromCR,
	)
	nicWriterAdapter := k8sadapter.NewWriterAdapter[*nicdom.Nic](
		client.Client,
		nick8s.NICGVR,
		logger,
		nick8s.NicToCR,
		nick8s.NicFromCR,
	)

	sdknetworkapi.HandlerWithOptions(
		&netrest.Handler{
			NetworkReader: netReaderAdapter,
			NetworkWriter: netWriterAdapter,
			SKUReader:     netSKUReaderAdapter,
			NicReader:     nicReaderAdapter,
			NicWriter:     nicWriterAdapter,
			Logger:        logger,
		},
		sdknetworkapi.StdHTTPServerOptions{
			BaseURL:          "/providers/seca.network",
			BaseRouter:       mux,
			Middlewares:      auth.ProviderMWs[sdknetworkapi.MiddlewareFunc](authenticator, checker, "seca.network", "/providers/seca.network", logger),
			ErrorHandlerFunc: nil,
		},
	)

	// Storage adapters
	bsReaderAdapter := k8sadapter.NewReaderAdapter[*bsdom.BlockStorage](
		client.Client,
		bsk8s.BlockStorageGVR,
		logger,
		bsk8s.BlockStorageFromCR,
	)
	bsWriterAdapter := k8sadapter.NewWriterAdapter[*bsdom.BlockStorage](
		client.Client,
		bsk8s.BlockStorageGVR,
		logger,
		bsk8s.BlockStorageToCR,
		bsk8s.BlockStorageFromCR,
	)
	skuReaderAdapter := k8sadapter.NewReaderAdapter[*skudom.StorageSKU](
		client.Client,
		skuk8s.StorageSKUGVR,
		logger,
		skuk8s.StorageSKUFromCR,
	)
	imgReaderAdapter := k8sadapter.NewReaderAdapter[*imgdom.Image](
		client.Client,
		imgk8s.ImageGVR,
		logger,
		imgk8s.ImageFromCR,
	)
	imgWriterAdapter := k8sadapter.NewWriterAdapter[*imgdom.Image](
		client.Client,
		imgk8s.ImageGVR,
		logger,
		imgk8s.ImageToCR,
		imgk8s.ImageFromCR,
	)

	sdkstorageapi.HandlerWithOptions(
		&storagerest.Handler{
			BlockStorageReader: bsReaderAdapter,
			BlockStorageWriter: bsWriterAdapter,
			ImageReader:        imgReaderAdapter,
			ImageWriter:        imgWriterAdapter,
			SKUReader:          skuReaderAdapter,
			Logger:             logger,
		},
		sdkstorageapi.StdHTTPServerOptions{
			BaseURL:          "/providers/seca.storage",
			BaseRouter:       mux,
			Middlewares:      auth.ProviderMWs[sdkstorageapi.MiddlewareFunc](authenticator, checker, "seca.storage", "/providers/seca.storage", logger),
			ErrorHandlerFunc: nil,
		},
	)

	// Workspace adapters
	wsWriterAdapter := k8sadapter.NewNamespaceManagingWriterAdapter[*wsdom.Workspace](
		client.Client,
		client.ClientSet,
		wsk8s.WorkspaceGVR,
		logger,
		wsk8s.WorkspaceToCR,
		wsk8s.WorkspaceFromCR,
	)
	wsReaderAdapter := k8sadapter.NewReaderAdapter[*wsdom.Workspace](
		client.Client,
		wsk8s.WorkspaceGVR,
		logger,
		wsk8s.WorkspaceFromCR,
	)

	sdkworkspaceapi.HandlerWithOptions(
		&wsrest.Handler{
			Reader: wsReaderAdapter,
			Writer: wsWriterAdapter,
			Logger: logger,
		},
		sdkworkspaceapi.StdHTTPServerOptions{
			BaseURL:          "/providers/seca.workspace",
			BaseRouter:       mux,
			Middlewares:      auth.ProviderMWs[sdkworkspaceapi.MiddlewareFunc](authenticator, checker, "seca.workspace", "/providers/seca.workspace", logger),
			ErrorHandlerFunc: nil,
		},
	)

	httpServer := httpserver.New(
		httpserver.Options{
			Addr:    addr,
			Handler: mux,
			Logger:  logger,
		},
	)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("failed to start regional API server", "error", err)
		log.Fatal(err, " - failed to start regional API server")
	}
}
