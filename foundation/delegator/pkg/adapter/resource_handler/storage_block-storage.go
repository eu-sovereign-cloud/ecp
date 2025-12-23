package resourcehandler

import (
	"context"
	"errors"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

/*
 *
 * NOTE: Code in this file is not complete nor ready for production usage.
 *       It's only for exploratory development.
 *
 */

type StorageBlockStorageCSPPlugin interface {
	Create(ctx context.Context, resource *regional.StorageBlockStorageDomain) error
	Delete(ctx context.Context, resource *regional.StorageBlockStorageDomain) error
	IncreaseSize(ctx context.Context, resource *regional.StorageBlockStorageDomain) error
	SetImage(ctx context.Context, resource *regional.StorageBlockStorageDomain) error
}

type StorageBlockStorageResourceHandler struct {
	port.GenericDelegatorResourceHandler[*regional.StorageBlockStorageDomain]
	repo   gateway_port.Repo[*regional.StorageBlockStorageDomain]
	plugin StorageBlockStorageCSPPlugin
}

func NewStorageBlockStorageResourceHandler(
	repo gateway_port.Repo[*regional.StorageBlockStorageDomain],
	plugin StorageBlockStorageCSPPlugin,
) *StorageBlockStorageResourceHandler {
	handler := &StorageBlockStorageResourceHandler{
		repo:   repo,
		plugin: plugin,
	}

	// Add admission rejection conditions
	handler.AddRejectionConditions(blockChangeSKU)
	handler.AddRejectionConditions(blockDecreaseSize)

	// Add create operation
	handler.AddOperations(
		port.NewResourceOperationWithPluginDelegate(
			giveConditionToCreate,
			plugin.Create,
			propagateCreateSucess,
			repo.Create,
			propagateFailure,
			port.NoopSetStateFunc[*regional.StorageBlockStorageDomain](),
		),
	)

	// Add delete operation
	handler.AddOperations(
		port.NewResourceOperationWithPluginDelegate(
			giveConditionToDelete,
			plugin.Delete,
			propagateDeleteSucess,
			repo.Delete,
			propagateFailure,
			repo.Update,
		),
	)

	// Add increase size operation
	handler.AddOperations(
		port.NewResourceOperationWithPluginDelegate(
			giveConditionToIncreaseSize,
			plugin.IncreaseSize,
			propagateIncreaseSizeSucess,
			repo.Update,
			propagateFailure,
			repo.Update,
		),
	)

	// Add set image operation
	handler.AddOperations(
		port.NewResourceOperationWithPluginDelegate(
			giveConditionToSetImage,
			plugin.SetImage,
			propagateSetImageSucess,
			repo.Update,
			propagateFailure,
			repo.Update,
		),
	)

	return handler
}

//
// Admission Rejection Conditions

func blockChangeSKU(_ context.Context, resource *regional.StorageBlockStorageDomain) error {
	if resource.Status.State != model.StateCreating && resource.Status.SkuType != resource.Spec.SkuType {
		return errors.New("changing storage sku type is not allowed")
	}

	return nil
}

func blockDecreaseSize(_ context.Context, resource *regional.StorageBlockStorageDomain) error {
	if resource.Status.State != model.StateCreating && resource.Spec.SizeGB < resource.Status.SizeGB {
		return errors.New("decrease storage size is not allowed")
	}

	return nil
}

//
// Common Operation Components

func propagateFailure(resource *regional.StorageBlockStorageDomain, err error) {
	resource.Status.SetError(err)
}

//
// Create Operation Components

func giveConditionToCreate(resource *regional.StorageBlockStorageDomain) bool {
	return resource.Status.State == model.StateCreating
}

func propagateCreateSucess(resource *regional.StorageBlockStorageDomain) {
	resource.Status.StorageBlockStorageSpec = resource.Spec
	resource.Status.SetAvtive()
}

//
// Delete Operation Components

func giveConditionToDelete(resource *regional.StorageBlockStorageDomain) bool {
	return resource.Status.State == model.StateDeleting
}

func propagateDeleteSucess(resource *regional.StorageBlockStorageDomain) {
	// NoOp!
}

//
// IncreaseSize Operation Components

func giveConditionToIncreaseSize(resource *regional.StorageBlockStorageDomain) bool {
	return resource.Status.State == model.StateUpdating && resource.Spec.SizeGB > resource.Status.SizeGB
}

func propagateIncreaseSizeSucess(resource *regional.StorageBlockStorageDomain) {
	resource.Status.SizeGB = resource.Spec.SizeGB
}

//
// SetImage Operation Components

func giveConditionToSetImage(resource *regional.StorageBlockStorageDomain) bool {
	return resource.Status.State == model.StateUpdating && resource.Spec.SourceImageID != resource.Status.SourceImageID
}

func propagateSetImageSucess(resource *regional.StorageBlockStorageDomain) {
	resource.Status.SourceImageID = resource.Spec.SourceImageID
}
