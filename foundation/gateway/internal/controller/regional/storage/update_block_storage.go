package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type UpdateBlockStorage struct {
	Logger           *slog.Logger
	BlockStorageRepo port.WriterRepo[*regional.BlockStorageDomain]
}

func (c UpdateBlockStorage) Do(
	ctx context.Context, domain *regional.BlockStorageDomain,
) (*regional.BlockStorageDomain, error) {
	result, err := c.BlockStorageRepo.Update(ctx, domain)
	if err != nil {
		return nil, err
	}
	return *result, nil
}
