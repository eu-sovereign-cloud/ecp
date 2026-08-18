package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	authv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/auth"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/httpserver"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/logger"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/metrics"

	authrest "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/frontend/rest"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	rak8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment/backend/kubernetes"
	rolek8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role/backend/kubernetes"
	rdom "github.com/eu-sovereign-cloud/ecp/resource/region/v1"
	rk8s "github.com/eu-sovereign-cloud/ecp/resource/region/v1/backend/kubernetes"
	regionrest "github.com/eu-sovereign-cloud/ecp/resource/region/v1/frontend/rest"
)

var (
	host       string
	port       string
	kubeconfig string

	globalAuthFlags auth.Flags
	globalKubeFlags kubeclient.ClientFlags
)

var globalAPIServerCMD = &cobra.Command{
	Use:     "globalapiserver",
	Aliases: []string{"global"},
	Short:   "The API server command starts the global server for the ECP application",
	Long:    `The API server command starts the global server for the ECP application`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := logger.New(os.Getenv("APP_ENV"))
		if err := startGlobal(logger, host+":"+port, kubeconfig); err != nil {
			logger.Error("global API server failed", slog.Any("error", err))
			os.Exit(1)
		}
	},
}

func init() {
	globalAPIServerCMD.Flags().StringVar(&kubeconfig, "kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "Path to kubeconfig file")
	globalAPIServerCMD.Flags().StringVar(&host, "host", "0.0.0.0", "Host to bind the server to")
	globalAPIServerCMD.Flags().StringVarP(&port, "port", "p", "8080", "Port to bind the server to")
	auth.RegisterFlags(globalAPIServerCMD, &globalAuthFlags)
	kubeclient.RegisterClientFlags(globalAPIServerCMD, &globalKubeFlags)
	rootCmd.AddCommand(globalAPIServerCMD)
}

// startGlobal starts the backend HTTP server on the given address.
func startGlobal(logger *slog.Logger, addr string, kubeconfigPath string) error {
	logger.Info("Starting global API server on", slog.Any("addr", addr))
	metrics.RegisterUpstreamObserver()

	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Warn("could not get in-cluster config, falling back to kubeconfig file", slog.Any("error", err))
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return fmt.Errorf("build kubeconfig %s: %w", kubeconfigPath, err)
		}
	}

	if err := globalKubeFlags.ApplyToConfig(config); err != nil {
		return fmt.Errorf("apply kube client flags: %w", err)
	}
	logger.Info("kube client rate limit",
		slog.Float64("kube_qps", float64(globalKubeFlags.QPS)),
		slog.Int("kube_burst", globalKubeFlags.Burst),
	)

	client, err := kubeclient.NewFromConfig(config)
	if err != nil {
		return fmt.Errorf("create kubeclient: %w", err)
	}

	// Create a shared mux for all global handlers.
	mux := http.NewServeMux()
	readiness := httpserver.NewReadiness()
	// Probes are unauthenticated and registered before provider routes so kubelet
	// can hit them while the process is still wiring (readyz stays 503 until Set).
	httpserver.RegisterProbes(mux, readiness, client.CheckAPIServer)

	// Metrics endpoint — unauthenticated, mounted outside provider HandlerWithOptions.
	mux.Handle("/metrics", metrics.Handler())

	// Authorization adapters (reused by both the CRUD handler and the RBAC checker).
	roleReaderAdapter := k8sadapter.NewReaderAdapter[*roledom.Role](
		client.Client,
		rolek8s.RoleGVR,
		logger,
		rolek8s.RoleFromCR,
	)
	// Role and RoleAssignment CRs live in the tenant namespace, and the global cluster has no
	// other writer that would provision it — so these two ensure it themselves. NoChildNamespace:
	// neither owns a namespace for children of its own.
	roleWriterAdapter := k8sadapter.NewNamespaceManagingWriterAdapter[*roledom.Role](
		client.Client,
		client.ClientSet,
		rolek8s.RoleGVR,
		logger,
		rolek8s.RoleToCR,
		rolek8s.RoleFromCR,
		k8sadapter.NoChildNamespace,
		nil,
	)
	roleAssignmentReaderAdapter := k8sadapter.NewReaderAdapter[*radom.RoleAssignment](
		client.Client,
		rak8s.RoleAssignmentGVR,
		logger,
		rak8s.RoleAssignmentFromCR,
	)
	roleAssignmentWriterAdapter := k8sadapter.NewNamespaceManagingWriterAdapter[*radom.RoleAssignment](
		client.Client,
		client.ClientSet,
		rak8s.RoleAssignmentGVR,
		logger,
		rak8s.RoleAssignmentToCR,
		rak8s.RoleAssignmentFromCR,
		k8sadapter.NoChildNamespace,
		nil,
	)

	// Build the authenticator and RBAC checker (both nil when --auth-enabled is not set).
	authenticator, checker, err := auth.Build(&globalAuthFlags, client.Client, roleReaderAdapter, roleAssignmentReaderAdapter, logger)
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

	// Region adapters and handler.
	regionv1.HandlerWithOptions(
		&regionrest.Handler{
			Repo:   k8sadapter.NewReaderAdapter[*rdom.Region](client.Client, rk8s.RegionGVR, logger, rk8s.RegionFromCR),
			Logger: logger,
		},
		regionv1.StdHTTPServerOptions{
			BaseURL:          rdom.RegionBaseURL,
			BaseRouter:       mux,
			Middlewares:      auth.ProviderMWs[regionv1.MiddlewareFunc](&globalAuthFlags, authenticator, checker, "seca.region", rdom.RegionBaseURL, logger),
			ErrorHandlerFunc: nil,
		},
	)

	// Authorization CRUD handler (Roles + RoleAssignments).
	authv1.HandlerWithOptions(
		&authrest.Handler{
			RoleReader:           roleReaderAdapter,
			RoleWriter:           roleWriterAdapter,
			RoleAssignmentReader: roleAssignmentReaderAdapter,
			RoleAssignmentWriter: roleAssignmentWriterAdapter,
			Logger:               logger,
		},
		authv1.StdHTTPServerOptions{
			BaseURL:          roledom.AuthorizationBaseURL,
			BaseRouter:       mux,
			Middlewares:      auth.ProviderMWs[authv1.MiddlewareFunc](&globalAuthFlags, authenticator, checker, "seca.authorization", roledom.AuthorizationBaseURL, logger),
			ErrorHandlerFunc: nil,
		},
	)

	httpServer := httpserver.New(httpserver.Options{
		Addr:    addr,
		Handler: metrics.HTTPMiddleware(mux),
		Logger:  logger,
	})

	// Open the readiness gate only after full wiring; Serve clears it on SIGTERM.
	readiness.Set(true)
	logger.Info("Global API server started successfully")
	if err := httpserver.Serve(ctx, httpServer, logger, readiness); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	logger.Info("Global API server shut down gracefully")
	return nil
}
