package regionalprovider

import (
	"context"
	"log/slog"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"

	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/network/skus/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
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

// NetworkController implements the NetworkProvider interface
type NetworkController struct {
	Logger  *slog.Logger
	SKURepo port.ResourceQueryRepository[*regional.NetworkSKUDomain]
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

	listParams := model.ListParams{
		Namespace: tenantID,
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}

	var domainSKUs []*regional.NetworkSKUDomain
	nextSkipToken, err := c.SKURepo.List(ctx, listParams, &domainSKUs)
	if err != nil {
		return nil, err
	}

	sdkNetworkSKUs := make([]sdkschema.NetworkSku, len(domainSKUs))
	for i := range domainSKUs {
		sdkNetworkSKUs[i] = *api.ToSDKNetworkSKU(domainSKUs[i])
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
	// scope by tenant namespace if needed
	domain.SetNamespace(tenantID)
	if err := c.SKURepo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return api.ToSDKNetworkSKU(domain), nil
}

func (n *NetworkController) ListPublicIps(ctx context.Context, tenantID, workspaceID string, params sdknetwork.ListPublicIpsParams) (*secapi.Iterator[sdkschema.PublicIp], error) {
	// TODO implement me
	n.Logger.Debug("implement me")
	panic("implement me")
}

func (n *NetworkController) GetPublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string) (sdkschema.PublicIp, error) {
	// TODO implement me
	n.Logger.Debug("implement me")
	panic("implement me")
}

func (n *NetworkController) CreateOrUpdatePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params sdknetwork.CreateOrUpdatePublicIpParams, req sdkschema.PublicIp) (*sdkschema.PublicIp, bool, error) {
	// TODO implement me
	n.Logger.Debug("implement me")
	panic("implement me")
}

func (n *NetworkController) DeletePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params sdknetwork.DeletePublicIpParams) error {
	// TODO implement me
	n.Logger.Debug("implement me")
	panic("implement me")
}
