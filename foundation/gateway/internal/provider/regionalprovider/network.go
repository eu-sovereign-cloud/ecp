package regionalprovider

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/client-go/rest"

	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/network/skus/v1"
	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/provider/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
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
	client *kubeclient.KubeClient
	logger *slog.Logger
}

// NewNetworkController creates a new NetworkController with a Kubernetes client.
func NewNetworkController(logger *slog.Logger, cfg *rest.Config) (*NetworkController, error) {
	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		logger.Error("failed to create kubeclient", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create kubeclient: %w", err)
	}

	return &NetworkController{
		client: client,
		logger: logger.With(slog.String("component", "NetworkController")),
	}, nil
}

func (c NetworkController) ListSKUs(ctx context.Context, tenantID string, params sdknetwork.ListSkusParams) (
	*sdknetwork.SkuIterator, error,
) {
	limit := validation.GetLimit(params.Limit)

	convert := common.Adapter(func(crdNetworkSKU skuv1.NetworkSKU) (sdkschema.NetworkSku, error) {
		return fromCRToSDKNetworkSKU(crdNetworkSKU), nil
	})
	opts := common.NewListOptions().Namespace(tenantID)
	if limit > 0 {
		opts.Limit(limit)
	}
	if params.SkipToken != nil {
		opts.SkipToken(*params.SkipToken)
	}
	if params.Labels != nil {
		opts.Selector(*params.Labels)
	}

	sdkNetworkSKUs, nextSkipToken, err := common.ListResources(ctx, c.client.Client, skuv1.NetworkSKUGVR, *c.logger, convert, opts)
	if err != nil {
		return nil, err
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
	convert := common.Adapter(func(crdNetworkSKU skuv1.NetworkSKU) (sdkschema.NetworkSku, error) {
		return fromCRToSDKNetworkSKU(crdNetworkSKU), nil
	})
	opts := common.NewGetOptions().Namespace(tenantID)
	sku, err := common.GetResource(ctx, c.client.Client, skuv1.NetworkSKUGVR, skuID, *c.logger, convert, opts)
	if err != nil {
		return nil, err
	}
	return &sku, nil
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

func fromCRToSDKNetworkSKU(crNetworkSKU skuv1.NetworkSKU) sdkschema.NetworkSku {
	sdkNetworkSKU := sdkschema.NetworkSku{
		Spec: &sdkschema.NetworkSkuSpec{
			Bandwidth: crNetworkSKU.Spec.Bandwidth,
			Packets:   crNetworkSKU.Spec.Packets,
		},
		Metadata: &sdkschema.SkuResourceMetadata{
			Name: crNetworkSKU.GetName(),
		},
	}
	return sdkNetworkSKU
}
