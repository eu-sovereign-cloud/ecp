package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/repository"
)

type SecurityGroupRepository = *repository.GenericRepository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList]

func NewSecurityGroupRepository(ctx context.Context, cli client.Client, ca crcache.Cache) SecurityGroupRepository {
	return repository.NewGenericRepository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList](ctx, cli, ca)
}
