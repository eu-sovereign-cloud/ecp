package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

type CreateBlockStorage struct {
	Logger           *slog.Logger
	BlockStorageRepo port.WriterRepo[*regional.BlockStorageDomain]
}

func (c CreateBlockStorage) Do(
	ctx context.Context, domain *regional.BlockStorageDomain,
) (*regional.BlockStorageDomain, error) {
	result, err := c.BlockStorageRepo.Create(ctx, domain)
	if err != nil {
		return nil, err
	}
	return *result, nil
}
