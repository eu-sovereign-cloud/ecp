package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/repository"
)

type SecurityRuleRepository = *repository.GenericRepository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList]

func NewSecurityRuleRepository(ctx context.Context, cli client.Client, ca crcache.Cache) SecurityRuleRepository {
	return repository.NewGenericRepository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList](ctx, cli, ca)
}
