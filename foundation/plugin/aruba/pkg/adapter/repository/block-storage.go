package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
)

type StorageRepository = *repository.GenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList]

func NewStorageRepository(ctx context.Context, cli client.Client, ca crcache.Cache) StorageRepository {
	return repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctx, cli, ca)
}
