package regionalprovider

import (
	"context"
	"fmt"
	"log/slog"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
	"k8s.io/client-go/rest"

	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/network/skus/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

const (
	NetworkBaseURL      = "/providers/seca.network"
	ProviderNetworkName = "seca.network/v1"
)

type NetworkSKUProvider interface {
	ListSKUs(ctx context.Context, tenantID string, params sdknetwork.ListSkusParams) (*sdknetwork.SkuIterator, error)
	GetSKU(ctx context.Context, tenantID, skuID string) (*sdkschema.NetworkSku, error)
}

type PublicIPProvider interface {
	ListPublicIps(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListPublicIpsParams) (*secapi.Iterator[sdkschema.PublicIp], error)
	GetPublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string) (sdkschema.PublicIp, error)
	CreateOrUpdatePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params sdknetwork.CreateOrUpdatePublicIpParams, req sdkschema.PublicIp) (*sdkschema.PublicIp, bool, error)
	DeletePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params sdknetwork.DeletePublicIpParams) error
}

type NetworkProvider interface {
	NetworkSKUProvider
	PublicIPProvider
}

var _ NetworkProvider = (*NetworkController)(nil) // Ensure NetworkController implements the NetworkProvider interface.

// NetworkController implements the NetworkProvider interface and provides methods to interact with the Network CRDs and XRDs in the Kubernetes cluster.
type NetworkController struct {
	client         *kubeclient.KubeClient
	logger         *slog.Logger
	networkSKURepo port.ResourceQueryRepository[*regional.NetworkSKUDomain]
}

// NewNetworkController creates a new NetworkController with a Kubernetes client.
func NewNetworkController(logger *slog.Logger, cfg *rest.Config) (*NetworkController, error) {
	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		logger.Error("failed to create kubeclient", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create kubeclient: %w", err)
	}

	convert := func(u unstructured.Unstructured) (*regional.NetworkSKUDomain, error) {
		var crdNetworkSKU skuv1.NetworkSKU
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &crdNetworkSKU); err != nil {
			return &regional.NetworkSKUDomain{}, err
		}
		return regional.FromCRToNetworkSKUDomain(crdNetworkSKU), nil
	}

	networkSKUAdapter := kubernetes.NewAdapter(
		client.Client,
		skuv1.NetworkSKUGVR,
		logger,
		convert,
	)

	return &NetworkController{
		client:         client,
		logger:         logger.With(slog.String("component", "NetworkController")),
		networkSKURepo: networkSKUAdapter,
	}, nil
}

func (c NetworkController) ListSKUs(ctx context.Context, tenantID string, params sdknetwork.ListSkusParams) (
	*sdknetwork.SkuIterator, error,
) {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	listParams := port.ListParams{
		Namespace: tenantID,
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}

	var domainSKUs []*regional.NetworkSKUDomain
	nextSkipToken, err := c.networkSKURepo.List(ctx, listParams, &domainSKUs)
	if err != nil {
		return nil, err
	}

	sdkNetworkSKUs := make([]sdkschema.NetworkSku, len(domainSKUs))
	for i := range domainSKUs {
		sdkNetworkSKUs[i] = *regional.ToSDKNetworkSKU(domainSKUs[i])
	}

	iterator := sdknetwork.SkuIterator{
		Items: sdkNetworkSKUs,
		Metadata: sdkschema.ResponseMetadata{
			Provider: ProviderNetworkName,
			Resource: skuv1.NetworkSKUResource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return &iterator, nil
}

func (c NetworkController) GetSKU(
	ctx context.Context, tenantID, skuID string,
) (*sdkschema.NetworkSku, error) {
	domain := &regional.NetworkSKUDomain{}
	domain.SetName(skuID)
	// scope by tenant namespace if needed (CRD is Namespaced per kubebuilder tag)
	domain.SetNamespace(tenantID)
	if err := c.networkSKURepo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return regional.ToSDKNetworkSKU(domain), nil
}

func (n *NetworkController) ListPublicIps(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListPublicIpsParams) (*secapi.Iterator[sdkschema.PublicIp], error) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *NetworkController) GetPublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string) (sdkschema.PublicIp, error) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *NetworkController) CreateOrUpdatePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params sdknetwork.CreateOrUpdatePublicIpParams, req sdkschema.PublicIp) (*sdkschema.PublicIp, bool, error) {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}

func (n *NetworkController) DeletePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params sdknetwork.DeletePublicIpParams) error {
	// TODO implement me
	n.logger.Debug("implement me")
	panic("implement me")
}
