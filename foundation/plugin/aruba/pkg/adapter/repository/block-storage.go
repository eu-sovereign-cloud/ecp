package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
)

type BlockStorageRepository = *repository.GenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList]

<<<<<<< HEAD
func NewBlockStorageRepository(ctx context.Context, cli client.Client, ca crcache.Cache) BlockStorageRepository {
	return repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctx, cli, ca)
=======
func NewStorageRepository(client client.Client, cache crcache.Cache) StorageRepository {
	return generic_repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](client, cache)
>>>>>>> e462b85 (feat: Use context with cancel inside handle method)
}
