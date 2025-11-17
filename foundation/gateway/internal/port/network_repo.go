package port

import (
	"context"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
)

type NetworkSKUProvider interface {
	ListSKUs(ctx context.Context, tenantID string, params network.ListSkusParams) (*network.SkuIterator, error)
	GetSKU(ctx context.Context, tenantID, skuID string) (*schema.NetworkSku, error)
}

type PublicIPProvider interface {
	ListPublicIps(ctx context.Context, tenantID, workspaceID string, params network.ListPublicIpsParams) (*secapi.Iterator[schema.PublicIp], error)
	GetPublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string) (schema.PublicIp, error)
	CreateOrUpdatePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params network.CreateOrUpdatePublicIpParams, req schema.PublicIp) (*schema.PublicIp, bool, error)
	DeletePublicIp(ctx context.Context, tenantID, workspaceID, publicIpID string, params network.DeletePublicIpParams) error
}

type NetworkProvider interface {
	NetworkSKUProvider
	PublicIPProvider
}
