package cmd

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/eu-sovereign-cloud/ecp/internal/httpserver"

	"github.com/eu-sovereign-cloud/ecp/internal/handler"
	"github.com/eu-sovereign-cloud/ecp/internal/logger"
	"github.com/eu-sovereign-cloud/ecp/internal/provider/globalprovider"
)

var (
	host       string
	port       string
	kubeconfig string
)

var globalAPIServerCMD = &cobra.Command{
	Use:     "globalapiserver",
	Aliases: []string{"global"},
	Short:   "The API server command starts the global server for the ECP application",
	Long:    `The API server command starts the global server for the ECP application`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := logger.New(os.Getenv("APP_ENV"))
		startGlobal(logger, host+":"+port, kubeconfig)
	},
}

func init() {
	globalAPIServerCMD.Flags().StringVar(&kubeconfig, "kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "Path to kubeconfig file")
	globalAPIServerCMD.Flags().StringVar(&host, "host", "0.0.0.0", "Host to bind the server to")
	globalAPIServerCMD.Flags().StringVarP(&port, "port", "p", "8080", "Port to bind the server to")
	rootCmd.AddCommand(globalAPIServerCMD)
}

// startGlobal starts the backend HTTP server on the given address.
func startGlobal(logger *slog.Logger, addr string, kubeconfigPath string) {
	logger.Info("Starting global API server on", slog.Any("addr", addr))

	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Warn("could not get in-cluster config, falling back to kubeconfig file", slog.Any("error", err))
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			logger.Error("failed to build kubeconfig", "path", kubeconfigPath, slog.Any("error", err))
			log.Fatal(err, " - failed to build kubeconfig")
		}
	}

	globalServer, err := globalprovider.NewController(logger, config)
	if err != nil {
		logger.Error("failed to create global server", slog.Any("error", err))
		log.Fatal(err, " - failed to create global server")
	}

	regionalHandler := handler.NewRegionHandler(logger, globalServer)
	regionHandler := region.HandlerWithOptions(regionalHandler, region.StdHTTPServerOptions{
		BaseURL:          "/providers/seca.region",
		BaseRouter:       nil,
		Middlewares:      nil,
		ErrorHandlerFunc: nil,
	})

	httpServer := httpserver.New(httpserver.Options{
		Addr:    addr,
		Handler: regionHandler,
		Logger:  logger,
	})

	logger.Info("Global API server started successfully")
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("failed to start global API server", slog.Any("error", err))
		log.Fatal(err, " - failed to start global API server")
	}
}
