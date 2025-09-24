package cmd

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	storage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/eu-sovereign-cloud/ecp/internal/handler"
	"github.com/eu-sovereign-cloud/ecp/internal/httpserver"
	"github.com/eu-sovereign-cloud/ecp/internal/logger"
	"github.com/eu-sovereign-cloud/ecp/internal/provider/regionalprovider"
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
	regionalApiServerCMD.Flags().StringVar(&regionalKubeconfig, "regionalKubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "Path to regional kubeconfig file",
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

	storageController, err := regionalprovider.NewController(logger, config)
	if err != nil {
		logger.Error("failed to create regional provider", slog.Any("error", err))
		log.Fatal(err, " - failed to create regional provider")
	}

	storageHandler := handler.NewStorageHandler(logger, storageController)
	logger.Info("Starting regional API server on", "addr", addr)

	httpServer := httpserver.New(
		httpserver.Options{
			Addr:    addr,
			Handler: storage.HandlerFromMuxWithBaseURL(storageHandler, nil, ""),
			Logger:  logger,
		},
	)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("failed to start regional API server", "error", err)
		log.Fatal(err, " - failed to start regional API server")
	}
}
