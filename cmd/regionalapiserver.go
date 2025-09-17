package cmd

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	compute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	"github.com/spf13/cobra"

	"github.com/eu-sovereign-cloud/ecp/internal/handler"
	"github.com/eu-sovereign-cloud/ecp/internal/logger"
	"github.com/eu-sovereign-cloud/ecp/internal/provider/regionalprovider"
)

var (
	regionalHost string
	regionalPort string
)

var regionalApiServerCMD = &cobra.Command{
	Use:   "regionalapiserver",
	Short: "The command starts the regional server for the ECP application",
	Long:  `The command starts the regional server for the ECP application`,
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

func startRegional(logger *slog.Logger, addr string) {
	computeHandler := handler.NewComputeHandler(regionalprovider.ComputeServer{})

	logger.Info("Starting API server on", "addr", addr)
	httpLogger := slog.NewLogLogger(logger.Handler(), slog.LevelInfo)
	// todo do this separately and use for server as well
	httpServer := &http.Server{
		Addr:         addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorLog:     httpLogger,
		Handler:      compute.HandlerFromMuxWithBaseURL(computeHandler, nil, ""),
	}

	if err := httpServer.ListenAndServe(); err != nil {
		logger.Error("failed to start global API server", slog.Any("error", err))
		log.Fatal(err, " - failed to start global API server")
	}
}
