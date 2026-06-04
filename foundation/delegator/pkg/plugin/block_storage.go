package plugin

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
)

type BlockStorage interface {
	Create(ctx context.Context, resource *regional.BlockStorageDomain) error
	Delete(ctx context.Context, resource *regional.BlockStorageDomain) error
	IncreaseSize(ctx context.Context, resource *regional.BlockStorageDomain) error
}
