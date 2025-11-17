package storage

import (
	"context"
	"log/slog"

	storage2 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

const (
	BaseURL             = "/providers/seca.storage"
	ProviderStorageName = "seca.storage/v1"
)

type Controller struct {
	Logger  *slog.Logger
	SKURepo port.ResourceQueryRepository[*regional.StorageSKUDomain]
}

const TenantLabelKey = "secapi.cloud/tenant-id"

func (c Controller) ListBlockStorages(
	ctx context.Context, tenantID, workspaceID string, params storage2.ListBlockStoragesParams,
) (*secapi.Iterator[schema.BlockStorage], error) {
	// TODO implement me
	panic("implement me")
}

func (c Controller) GetBlockStorage(
	ctx context.Context, tenantID, workspaceID, storageID string,
) (*schema.BlockStorage, error) {
	// TODO implement me
	panic("implement me")
}

func (c Controller) CreateOrUpdateBlockStorage(
	ctx context.Context, tenantID, workspaceID, storageID string,
	params storage2.CreateOrUpdateBlockStorageParams, req schema.BlockStorage,
) (*schema.BlockStorage, bool, error) {
	// TODO implement me
	panic("implement me")
}

func (c Controller) DeleteBlockStorage(
	ctx context.Context, tenantID, workspaceID, storageID string, params storage2.DeleteBlockStorageParams,
) error {
	// TODO implement me
	panic("implement me")
}
