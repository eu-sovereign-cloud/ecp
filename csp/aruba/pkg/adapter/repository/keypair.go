package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/repository"
)

type KeyPairRepository = *repository.GenericRepository[*v1alpha1.KeyPair, *v1alpha1.KeyPairList]

func NewKeyPairRepository(ctx context.Context, cli client.Client, ca crcache.Cache) KeyPairRepository {
	return repository.NewGenericRepository[*v1alpha1.KeyPair, *v1alpha1.KeyPairList](ctx, cli, ca)
}
