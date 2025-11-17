package storage

import (
	"context"

	storage2 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
)

func (c Controller) CreateOrUpdateImage(ctx context.Context, tenantID, imageID string, params storage2.CreateOrUpdateImageParams, req schema.Image) (*schema.Image, bool, error) {
	// TODO implement me
	panic("implement me")
}

func (c Controller) GetImage(ctx context.Context, tenantID, imageID string) (*schema.Image, error) {
	// TODO implement me
	panic("implement me")
}

func (c Controller) DeleteImage(ctx context.Context, tenantID, imageID string, params storage2.DeleteImageParams) error {
	// TODO implement me
	panic("implement me")
}

func (c Controller) ListImages(ctx context.Context, tenantID string, params storage2.ListImagesParams) (*secapi.Iterator[schema.Image], error) {
	// TODO implement me
	panic("implement me")
}
