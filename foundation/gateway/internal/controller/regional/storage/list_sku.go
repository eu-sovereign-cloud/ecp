package storage

import (
	"context"
	"log/slog"

	v1 "github.com/eu-sovereign-cloud/ecp/foundation/api/block-storage/skus/v1"
	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/storage"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

const (
	BaseURL             = "/providers/seca.storage"
	ProviderStorageName = "seca.storage/v1"
)

type ListSKUs struct {
	Logger  *slog.Logger
	SKURepo port.ResourceQueryRepository[*regional.StorageSKUDomain]
}

func (c ListSKUs) Do(ctx context.Context, tenantID string, params storagev1.ListSkusParams) (
	*storagev1.SkuIterator, error,
) {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	listParams := model.ListParams{
		Namespace: tenantID,
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
	var domainSKUs []*regional.StorageSKUDomain
	nextSkipToken, err := c.SKURepo.List(ctx, listParams, &domainSKUs)
	if err != nil {
		return nil, err
	}

	// convert to sdk slice
	sdkSKUs := make([]schema.StorageSku, len(domainSKUs))
	for i := range domainSKUs {
		mapped := storage.SkuToApi(domainSKUs[i])
		sdkSKUs[i] = *mapped
	}

	iterator := storagev1.SkuIterator{
		Items: sdkSKUs,
		Metadata: schema.ResponseMetadata{
			Provider: ProviderStorageName,
			Resource: v1.StorageSKUResource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return &iterator, nil
}
