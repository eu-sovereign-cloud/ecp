package secapi

import (
	"context"
	"fmt"
	"net/http"

	storage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"k8s.io/utils/ptr"
)

type StorageV1 struct {
	API
	storage storage.ClientWithResponsesInterface
}

// Storage Sku

func (api *StorageV1) ListSkus(ctx context.Context, tid TenantID) (*Iterator[schema.StorageSku], error) {
	iter := Iterator[schema.StorageSku]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.StorageSku, *string, error) {
			resp, err := api.storage.ListSkusWithResponse(ctx, schema.TenantPathParam(tid), &storage.ListSkusParams{
				Accept: ptr.To(storage.ListSkusParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *StorageV1) GetSku(ctx context.Context, tref TenantReference) (*schema.StorageSku, error) {
	if err := tref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.storage.GetSkuWithResponse(ctx, schema.TenantPathParam(tref.Tenant), tref.Name, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

// Block Storage

func (api *StorageV1) ListBlockStorages(ctx context.Context, tid TenantID, wid WorkspaceID) (*Iterator[schema.BlockStorage], error) {
	iter := Iterator[schema.BlockStorage]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.BlockStorage, *string, error) {
			resp, err := api.storage.ListBlockStoragesWithResponse(ctx, schema.TenantPathParam(tid), schema.WorkspacePathParam(wid), &storage.ListBlockStoragesParams{
				Accept: ptr.To(storage.ListBlockStoragesParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *StorageV1) GetBlockStorage(ctx context.Context, wref WorkspaceReference) (*schema.BlockStorage, error) {
	if err := wref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.storage.GetBlockStorageWithResponse(ctx, schema.TenantPathParam(wref.Tenant), schema.WorkspacePathParam(wref.Workspace), wref.Name, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *StorageV1) CreateOrUpdateBlockStorageWithParams(ctx context.Context, block *schema.BlockStorage, params *storage.CreateOrUpdateBlockStorageParams) (*schema.BlockStorage, error) {
	if err := api.validateWorkspaceMetadata(block.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.storage.CreateOrUpdateBlockStorageWithResponse(ctx, schema.TenantPathParam(block.Metadata.Tenant), schema.WorkspacePathParam(block.Metadata.Workspace), block.Metadata.Name, params, *block, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *StorageV1) CreateOrUpdateBlockStorage(ctx context.Context, block *schema.BlockStorage) (*schema.BlockStorage, error) {
	return api.CreateOrUpdateBlockStorageWithParams(ctx, block, nil)
}

func (api *StorageV1) DeleteBlockStorageWithParams(ctx context.Context, block *schema.BlockStorage, params *storage.DeleteBlockStorageParams) error {
	if err := api.validateWorkspaceMetadata(block.Metadata); err != nil {
		return err
	}

	resp, err := api.storage.DeleteBlockStorageWithResponse(ctx, block.Metadata.Tenant, block.Metadata.Workspace, block.Metadata.Name, params, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *StorageV1) DeleteBlockStorage(ctx context.Context, block *schema.BlockStorage) error {
	return api.DeleteBlockStorageWithParams(ctx, block, nil)
}

// Image

func (api *StorageV1) ListImages(ctx context.Context, tid TenantID) (*Iterator[schema.Image], error) {
	iter := Iterator[schema.Image]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.Image, *string, error) {
			resp, err := api.storage.ListImagesWithResponse(ctx, schema.TenantPathParam(tid), &storage.ListImagesParams{
				Accept: ptr.To(storage.ListImagesParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *StorageV1) GetImage(ctx context.Context, tref TenantReference) (*schema.Image, error) {
	if err := tref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.storage.GetImageWithResponse(ctx, schema.TenantPathParam(tref.Tenant), tref.Name, api.loadRequestHeaders, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *StorageV1) CreateOrUpdateImageWithParams(ctx context.Context, image *schema.Image, params *storage.CreateOrUpdateImageParams) (*schema.Image, error) {
	if err := api.validateRegionalMetadata(image.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.storage.CreateOrUpdateImageWithResponse(ctx, schema.TenantPathParam(image.Metadata.Tenant), image.Metadata.Name, params, *image, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *StorageV1) CreateOrUpdateImage(ctx context.Context, image *schema.Image) (*schema.Image, error) {
	return api.CreateOrUpdateImageWithParams(ctx, image, nil)
}

func (api *StorageV1) DeleteImageWithParams(ctx context.Context, image *schema.Image, params *storage.DeleteImageParams) error {
	if err := api.validateRegionalMetadata(image.Metadata); err != nil {
		return err
	}

	resp, err := api.storage.DeleteImageWithResponse(ctx, image.Metadata.Tenant, image.Metadata.Name, params, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *StorageV1) DeleteImage(ctx context.Context, image *schema.Image) error {
	return api.DeleteImageWithParams(ctx, image, nil)
}

func (api *StorageV1) BuildReferenceURN(urn string) (*schema.Reference, error) {
	urnRef := schema.ReferenceURN(urn)

	ref := &schema.Reference{}
	if err := ref.FromReferenceURN(urnRef); err != nil {
		return nil, fmt.Errorf("error building referenceURN from URN %s: %s", urn, err)
	}

	return ref, nil
}

func (api *StorageV1) validateRegionalMetadata(metadata *schema.RegionalResourceMetadata) error {
	if metadata == nil {
		return ErrNoMetatada
	}

	if metadata.Tenant == "" {
		return ErrNoMetatadaTenant
	}

	return nil
}

func (api *StorageV1) validateWorkspaceMetadata(metadata *schema.RegionalWorkspaceResourceMetadata) error {
	if metadata == nil {
		return ErrNoMetatada
	}

	if metadata.Tenant == "" {
		return ErrNoMetatadaTenant
	}

	if metadata.Workspace == "" {
		return ErrNoMetatadaWorkspace
	}

	return nil
}

func newStorageV1(client *RegionalClient, storageUrl string) (*StorageV1, error) {
	storage, err := storage.NewClientWithResponses(storageUrl)
	if err != nil {
		return nil, err
	}

	return &StorageV1{API: API{authToken: client.authToken}, storage: storage}, nil
}
