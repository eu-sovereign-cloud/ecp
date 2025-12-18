package cmd

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/regional/workspace"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/httpserver"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/logger"
	regionalhandler "github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/service/handler/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
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

	httpServer := httpserver.New(
		httpserver.Options{
			Addr: addr,
			// TODO: Merge storage, workspace and other regional handlers into a single one. For now, only workspace is enabled.
			// Handler: sdkstorageapi.HandlerWithOptions(
			// 	regionalhandler.Storage{
			// 		ListSKUs: &storage.ListSKUs{
			// 			Logger: logger,
			// 			SKURepo: kubernetes.NewReaderAdapter(
			// 				client.Client,
			// 				skuv1.SKUGVR,
			// 				logger,
			// 				kubernetes.MapCRToStorageSKUDomain,
			// 			),
			// 		},
			// 		GetSKU: &storage.GetSKU{
			// 			Logger: logger,
			// 			SKURepo: kubernetes.NewReaderAdapter(
			// 				client.Client,
			// 				skuv1.SKUGVR,
			// 				logger,
			// 				kubernetes.MapCRToStorageSKUDomain,
			// 			),
			// 		},
			// 		Logger: logger,
			// 	}, sdkstorageapi.StdHTTPServerOptions{
			// 		BaseURL:          apistorage.BaseURL,
			// 		BaseRouter:       nil,
			// 		Middlewares:      nil,
			// 		ErrorHandlerFunc: nil,
			// 	},
			// ),
			Handler: sdkworkspace.HandlerWithOptions(
				regionalhandler.Workspace{
					List: &workspace.ListWorkspaces{
						Logger: logger,
						Repo: kubernetes.NewReaderAdapter(
							client.Client,
							workspacev1.GroupVersionResource,
							logger,
							kubernetes.MapCRToWorkspaceDomain,
						),
					},
					Get: &workspace.GetWorkspace{
						Logger: logger,
						Repo: kubernetes.NewReaderAdapter(
							client.Client,
							workspacev1.GroupVersionResource,
							logger,
							kubernetes.MapCRToWorkspaceDomain,
						),
					},
					Create: &workspace.CreateWorkspace{
						Logger: logger,
						Repo: kubernetes.NewWriterAdapter(
							client.Client,
							workspacev1.GroupVersionResource,
							logger,
							kubernetes.MapWorkspaceDomainToCR,
							kubernetes.MapCRToWorkspaceDomain,
						),
					},
					Logger: logger,
				}, sdkworkspace.StdHTTPServerOptions{
					BaseURL:          regional.WorkspaceBaseURL, // TODO: dummy value, should retrieve actual value from runtime config
					BaseRouter:       nil,
					Middlewares:      nil,
					ErrorHandlerFunc: nil,
				},
			),
			Logger: logger,
		},
	)

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("failed to start regional API server", "error", err)
		log.Fatal(err, " - failed to start regional API server")
	}
}
