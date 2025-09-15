package cmd

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	compute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	"github.com/spf13/cobra"

	"github.com/eu-sovereign-cloud/ecp/internal/handler"
	"github.com/eu-sovereign-cloud/ecp/internal/logger"
	"github.com/eu-sovereign-cloud/ecp/internal/provider/regionalprovider"
	"github.com/eu-sovereign-cloud/ecp/internal/server"
)

var (
	regionalHost string
	regionalPort string
)

var regionalApiServerCMD = &cobra.Command{
	Use:     "regionalapiserver",
	Aliases: []string{"regional"},
	Short:   "The command starts the regional server for the ECP application",
	Long:    `The command starts the regional server for the ECP application`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := logger.New(os.Getenv("APP_ENV"))
		startRegional(logger, regionalHost+":"+regionalPort)
	},
}

func init() {
	regionalApiServerCMD.Flags().StringVar(&regionalHost, "regionalHost", "0.0.0.0", "Host to bind the server to")
	regionalApiServerCMD.Flags().StringVarP(&regionalPort, "regionalPort", "p", "8080", "Port to bind the server to")
	rootCmd.AddCommand(regionalApiServerCMD)
}

// startRegional starts the backend HTTP server on the given address.
func startRegional(logger *slog.Logger, addr string) {
	computeHandler := handler.NewComputeHandler(regionalprovider.ComputeServer{})
	regionHandler := compute.HandlerFromMuxWithBaseURL(computeHandler, nil, "")

	logger.Info("Starting regional API server on", "addr", addr)

	httpServer := server.NewHTTPServer(addr, regionHandler, logger)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("failed to start regional API server", "error", err)
		log.Fatal(err, " - failed to start regional API server")
	}
}
