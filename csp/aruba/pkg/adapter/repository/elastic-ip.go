package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/repository"
)

type ElasticIPRepository = *repository.GenericRepository[*v1alpha1.ElasticIP, *v1alpha1.ElasticIPList]

func NewElasticIPRepository(ctx context.Context, cli client.Client, ca crcache.Cache) ElasticIPRepository {
	return repository.NewGenericRepository[*v1alpha1.ElasticIP, *v1alpha1.ElasticIPList](ctx, cli, ca)
}
