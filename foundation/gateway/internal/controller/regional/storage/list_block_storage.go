package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type ListBlockStorages struct {
	Logger           *slog.Logger
	BlockStorageRepo port.ReaderRepo[*regional.BlockStorageDomain]
}

func (c ListBlockStorages) Do(ctx context.Context, params model.ListParams) (
	[]*regional.BlockStorageDomain, *string, error,
) {
	var domainBlockStorages []*regional.BlockStorageDomain
	nextSkipToken, err := c.BlockStorageRepo.List(ctx, params, &domainBlockStorages)
	if err != nil {
		return nil, nil, err
	}

	return domainBlockStorages, nextSkipToken, nil
}
