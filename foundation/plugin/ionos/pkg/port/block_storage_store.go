package port

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

type BlockStorageStore interface {
	Create(ctx context.Context, domain *regional.BlockStorageDomain) error
	Delete(ctx context.Context, domain *regional.BlockStorageDomain) error
	IncreaseSize(ctx context.Context, domain *regional.BlockStorageDomain) error
}
