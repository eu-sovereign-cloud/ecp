package plugin

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

type BlockStorage interface {
	Create(ctx context.Context, resource *regional.BlockStorageDomain) error
	Delete(ctx context.Context, resource *regional.BlockStorageDomain) error
	IncreaseSize(ctx context.Context, resource *regional.BlockStorageDomain) error
}

var _ port.DelegatedFunc[*regional.BlockStorageDomain] = ((BlockStorage)(nil)).Create
var _ port.DelegatedFunc[*regional.BlockStorageDomain] = ((BlockStorage)(nil)).Delete
var _ port.DelegatedFunc[*regional.BlockStorageDomain] = ((BlockStorage)(nil)).IncreaseSize
