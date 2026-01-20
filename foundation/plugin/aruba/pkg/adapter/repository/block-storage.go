package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
)

type BlockStorageRepository = *repository.GenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList]

func NewBlockStorageRepository(ctx context.Context, cli client.Client, ca crcache.Cache) BlockStorageRepository {
	return repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctx, cli, ca)
}
