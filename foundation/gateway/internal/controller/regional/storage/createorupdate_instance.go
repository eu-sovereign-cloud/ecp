package storage

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type CreateOrUpdateInstance struct {
	Logger      *slog.Logger
	StorageRepo port.Repo[*regional.BlockStorageDomain]
}

func (c CreateOrUpdateInstance) Do(
	ctx context.Context, domain *regional.BlockStorageDomain,
) (*regional.BlockStorageDomain, error) {
	// Try to load the existing resource
	existing := &regional.BlockStorageDomain{}
	existing.SetName(domain.GetName())
	existing.SetNamespace(domain.GetNamespace())

	err := c.StorageRepo.Load(ctx, &existing)
	if err != nil {
		// If not found, create it
		if errors.Is(err, model.ErrNotFound) {
			if err := c.StorageRepo.Create(ctx, domain); err != nil {
				c.Logger.ErrorContext(ctx, "failed to create block storage", slog.Any("error", err))
				return nil, err
			}
			return domain, nil
		}
		c.Logger.ErrorContext(ctx, "failed to load block storage", slog.Any("error", err))
		return nil, err
	}

	// Resource exists, update it
	// Merge the spec from the request with the existing resource
	existing.Spec = domain.Spec
	existing.Labels = domain.Labels
	version, _ := strconv.Atoi(domain.ResourceVersion)
	version++
	existing.ResourceVersion = strconv.Itoa(version)

	if err := c.StorageRepo.Update(ctx, existing); err != nil {
		c.Logger.ErrorContext(ctx, "failed to update block storage", slog.Any("error", err))
		return nil, err
	}

	return existing, nil
}
