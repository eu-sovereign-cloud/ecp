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
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	"github.com/eu-sovereign-cloud/ecp/framework/frontend/httpserver"
	"github.com/eu-sovereign-cloud/ecp/framework/frontend/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/framework/frontend/logger"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"

	netskudom "github.com/eu-sovereign-cloud/ecp/resources/regional/network/network-skus/v1"
	netskuk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/network/network-skus/v1/backend/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1"
	netk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1/backend/kubernetes"
	netrest "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1/frontend/rest"

	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/backend/kubernetes"
	storagerest "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/frontend/rest"
	skudom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/storage-skus/v1"
	skuk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/storage-skus/v1/backend/kubernetes"

	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1/backend/kubernetes"
	wsrest "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1/frontend/rest"
)

var (
	region             string
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
	rootCmd.AddCommand(regionalApiServerCMD)
}

// computeStub is a local stub implementing sdkcomputeapi.ServerInterface with all methods returning 501.
type computeStub struct {
	logger *slog.Logger
}

var _ sdkcomputeapi.ServerInterface = (*computeStub)(nil)

func (c *computeStub) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkcomputeapi.ListSkusParams) {
	c.logger.DebugContext(r.Context(), "ListSkus not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (c *computeStub) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	c.logger.DebugContext(r.Context(), "GetSku not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (c *computeStub) ListInstances(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdkcomputeapi.ListInstancesParams) {
	c.logger.DebugContext(r.Context(), "ListInstances not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (c *computeStub) DeleteInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcomputeapi.DeleteInstanceParams) {
	c.logger.DebugContext(r.Context(), "DeleteInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (c *computeStub) GetInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	c.logger.DebugContext(r.Context(), "GetInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (c *computeStub) CreateOrUpdateInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcomputeapi.CreateOrUpdateInstanceParams) {
	c.logger.DebugContext(r.Context(), "CreateOrUpdateInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (c *computeStub) RestartInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcomputeapi.RestartInstanceParams) {
	c.logger.DebugContext(r.Context(), "RestartInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (c *computeStub) StartInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcomputeapi.StartInstanceParams) {
	c.logger.DebugContext(r.Context(), "StartInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (c *computeStub) StopInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcomputeapi.StopInstanceParams) {
	c.logger.DebugContext(r.Context(), "StopInstance not implemented")
	w.WriteHeader(http.StatusNotImplemented)
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

	// Create a shared mux for all regional handlers
	mux := http.NewServeMux()

	// Compute (stub — not yet implemented)
	sdkcomputeapi.HandlerWithOptions(
		&computeStub{logger: logger},
		sdkcomputeapi.StdHTTPServerOptions{
			BaseURL:          "/providers/seca.compute",
			BaseRouter:       mux,
			Middlewares:      nil,
			ErrorHandlerFunc: nil,
		},
	)

	// Network adapters
	netReaderAdapter := k8sadapter.NewReaderAdapter[*netdom.Network](
		client.Client,
		netk8s.NetworkGVR,
		logger,
		netk8s.MapCRToNetworkDomain,
	)
	netWriterAdapter := k8sadapter.NewWriterAdapter[*netdom.Network](
		client.Client,
		netk8s.NetworkGVR,
		logger,
		netk8s.MapNetworkDomainToCR,
		netk8s.MapCRToNetworkDomain,
	)
	netSKUReaderAdapter := k8sadapter.NewReaderAdapter[*netskudom.NetworkSKU](
		client.Client,
		netskuk8s.NetworkSKUGVR,
		logger,
		netskuk8s.MapCRToNetworkSKUDomain,
	)

	sdknetworkapi.HandlerWithOptions(
		&netrest.Handler{
			NetworkReader: netReaderAdapter,
			NetworkWriter: netWriterAdapter,
			SKUReader:     netSKUReaderAdapter,
			Logger:        logger,
		},
		sdknetworkapi.StdHTTPServerOptions{
			BaseURL:          "/providers/seca.network",
			BaseRouter:       mux,
			Middlewares:      nil,
			ErrorHandlerFunc: nil,
		},
	)

	// Storage adapters
	bsReaderAdapter := k8sadapter.NewReaderAdapter[*bsdom.BlockStorage](
		client.Client,
		bsk8s.BlockStorageGVR,
		logger,
		bsk8s.MapCRToBlockStorageDomain,
	)
	bsWriterAdapter := k8sadapter.NewWriterAdapter[*bsdom.BlockStorage](
		client.Client,
		bsk8s.BlockStorageGVR,
		logger,
		bsk8s.MapBlockStorageDomainToCR,
		bsk8s.MapCRToBlockStorageDomain,
	)
	skuReaderAdapter := k8sadapter.NewReaderAdapter[*skudom.StorageSKU](
		client.Client,
		skuk8s.StorageSKUGVR,
		logger,
		skuk8s.MapCRToStorageSKUDomain,
	)

	sdkstorageapi.HandlerWithOptions(
		&storagerest.Handler{
			BlockStorageReader: bsReaderAdapter,
			BlockStorageWriter: bsWriterAdapter,
			SKUReader:          skuReaderAdapter,
			Logger:             logger,
		},
		sdkstorageapi.StdHTTPServerOptions{
			BaseURL:          "/providers/seca.storage",
			BaseRouter:       mux,
			Middlewares:      nil,
			ErrorHandlerFunc: nil,
		},
	)

	// Workspace adapters
	wsWriterAdapter := k8sadapter.NewNamespaceManagingWriterAdapter[*wsdom.Workspace](
		client.Client,
		client.ClientSet,
		wsk8s.WorkspaceGVR,
		logger,
		wsk8s.MapWorkspaceDomainToCR,
		wsk8s.MapCRToWorkspaceDomain,
	)
	wsReaderAdapter := k8sadapter.NewReaderAdapter[*wsdom.Workspace](
		client.Client,
		wsk8s.WorkspaceGVR,
		logger,
		wsk8s.MapCRToWorkspaceDomain,
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
