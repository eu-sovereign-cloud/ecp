package cmd

import (
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

	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/block-storages/v1"
	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/skus/v1"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/regional/storage"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/regional/workspace"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/httpserver"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/logger"
	regionalhandler "github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/service/handler/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	apicompute "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/compute"
	apinetwork "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/network"
	apistorage "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/storage"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

var (
	regionalHost       string
	regionalPort       string
	regionalKubeconfig string
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
		&regionalHost, "regionalHost", "0.0.0.0", "Host to bind the server to",
	)
	regionalApiServerCMD.Flags().StringVarP(
		&regionalPort, "regionalPort", "p", "8080", "Port to bind the server to",
	)
	regionalApiServerCMD.Flags().StringVar(
		&regionalKubeconfig, "kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"),
		"Path to regional kubeconfig",
	)
	rootCmd.AddCommand(regionalApiServerCMD)
}

// startRegional starts the backend HTTP server on the given address.
func startRegional(logger *slog.Logger, addr string, kubeconfigPath string) {
	logger.Info("Starting regional API server on", slog.Any("addr", addr))

	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Warn(
			"could not get in-cluster config, falling back to kubeconfig file",
			slog.Any("error", err),
		)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			logger.Error(
				"failed to build kubeconfig", "path", kubeconfigPath, slog.Any("error", err),
			)
			log.Fatal(err, " - failed to build kubeconfig")
		}
	}

	client, err := kubeclient.NewFromConfig(config)
	if err != nil {
		logger.Error("failed to create kubeclient", slog.Any("error", err))
		log.Fatal(err, " - failed to create kubeclient")
	}

	// Create a shared mux for all regional handlers
	mux := http.NewServeMux()

	sdkcomputeapi.HandlerWithOptions(regionalhandler.Compute{},
		sdkcomputeapi.StdHTTPServerOptions{BaseURL: apicompute.BaseURL, BaseRouter: mux, Middlewares: nil, ErrorHandlerFunc: nil})
	sdknetworkapi.HandlerWithOptions(regionalhandler.Network{},
		sdknetworkapi.StdHTTPServerOptions{BaseURL: apinetwork.BaseURL, BaseRouter: mux, Middlewares: nil, ErrorHandlerFunc: nil})
	// Block storage writer adapter
	blockStorageWriterAdapter := kubernetes.NewWriterAdapter(
		client.Client,
		blockstoragev1.BlockStorageGVR,
		logger,
		kubernetes.MapBlockStorageDomainToCR,
		kubernetes.MapCRToBlockStorageDomain,
	)
	skuReaderAdapter := kubernetes.NewReaderAdapter(
		client.Client,
		skuv1.SKUGVR,
		logger,
		kubernetes.MapCRToStorageSKUDomain,
	)
	storageReaderAdapter := kubernetes.NewReaderAdapter(
		client.Client,
		blockstoragev1.BlockStorageGVR,
		logger,
		kubernetes.MapCRToBlockStorageDomain,
	)
	// Register storage handler
	sdkstorageapi.HandlerWithOptions(
		regionalhandler.Storage{
			ListSKUs: &storage.ListSKUs{
				Logger:  logger,
				SKURepo: skuReaderAdapter,
			},
			GetSKU: &storage.GetSKU{
				Logger:  logger,
				SKURepo: skuReaderAdapter,
			},
			ListStorages: &storage.ListBlockStorages{
				Logger:           logger,
				BlockStorageRepo: storageReaderAdapter,
			},
			GetStorage: &storage.GetBlockStorage{
				Logger:           logger,
				BlockStorageRepo: storageReaderAdapter,
			},
			CreateBlockStorage: &storage.CreateBlockStorage{
				Logger:           logger,
				BlockStorageRepo: blockStorageWriterAdapter,
			},
			UpdateBlockStorage: &storage.UpdateBlockStorage{
				Logger:           logger,
				BlockStorageRepo: blockStorageWriterAdapter,
			},
			DeleteStorage: &storage.DeleteBlockStorage{
				Logger:           logger,
				BlockStorageRepo: blockStorageWriterAdapter,
			},
			Logger: logger,
		}, sdkstorageapi.StdHTTPServerOptions{
			BaseURL:          apistorage.BaseURL,
			BaseRouter:       mux,
			Middlewares:      nil,
			ErrorHandlerFunc: nil,
		},
	)

	// Workspace writer adapter that also manages namespace lifecycle
	// workspaceWriterAdapter := kubernetes.NewNamespaceManagingWriterAdapter(
	// 	client.Client,
	// 	client.ClientSet,
	// 	workspacev1.WorkspaceGVR,
	// 	logger,
	// 	kubernetes.MapWorkspaceDomainToCR,
	// 	kubernetes.MapCRToWorkspaceDomain,
	// )

	// Workspace reader adapter
	workspaceWriterAdapter := kubernetes.NewWriterAdapter(
		client.Client,
		workspacev1.WorkspaceGVR,
		logger,
		kubernetes.MapWorkspaceDomainToCR,
		kubernetes.MapCRToWorkspaceDomain,
	)

	// Workspace reader adapter
	workspaceReaderAdapter := kubernetes.NewReaderAdapter(
		client.Client,
		workspacev1.WorkspaceGVR,
		logger,
		kubernetes.MapCRToWorkspaceDomain,
	)

	// Register workspace handler
	sdkworkspaceapi.HandlerWithOptions(
		regionalhandler.Workspace{
			Logger: logger,
			Create: &workspace.CreateWorkspace{
				Logger: logger,
				Repo:   workspaceWriterAdapter,
			},
			Update: &workspace.UpdateWorkspace{
				Logger: logger,
				Repo:   workspaceWriterAdapter,
			},
			Delete: &workspace.DeleteWorkspace{
				Logger: logger,
				Repo:   workspaceWriterAdapter,
			},
			List: &workspace.ListWorkspace{
				Logger: logger,
				Repo:   workspaceReaderAdapter,
			},
			Get: &workspace.GetWorkspace{
				Logger: logger,
				Repo:   workspaceReaderAdapter,
			},
		}, sdkworkspaceapi.StdHTTPServerOptions{
			BaseURL:          regional.WorkspaceBaseURL,
			BaseRouter:       mux,
			Middlewares:      nil,
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
