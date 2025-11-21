package storage

import (
	v1 "github.com/eu-sovereign-cloud/ecp/foundation/api/block-storage/skus/v1"
	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

const (
	BaseURL             = "/providers/seca.storage"
	ProviderStorageName = "seca.storage/v1"
)

// SkuToApi converts a StorageSKUDomain to its SDK representation.
func SkuToApi(domain *regional.StorageSKUDomain) *sdkschema.StorageSku {
	return &sdkschema.StorageSku{
		Metadata: &sdkschema.SkuResourceMetadata{
			Name: domain.Name, // no namespace?
		},
		Spec: &sdkschema.StorageSkuSpec{
			Iops:          int(domain.Spec.Iops),
			MinVolumeSize: int(domain.Spec.MinVolumeSize),
			Type:          sdkschema.StorageSkuSpecType(domain.Spec.Type),
		},
	}
}

func ListParamsFromAPI(params sdkstorage.ListSkusParams, namespace string) model.ListParams {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	return model.ListParams{
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
		Namespace: namespace,
	}
}

func SKUDomainToAPIIterator(domainSKUs []*regional.StorageSKUDomain, nextSkipToken *string) *sdkstorage.SkuIterator {
	sdkSKUs := make([]sdkschema.StorageSku, len(domainSKUs))
	for i := range domainSKUs {
		mapped := SkuToApi(domainSKUs[i])
		sdkSKUs[i] = *mapped
	}

	iterator := &sdkstorage.SkuIterator{
		Items: sdkSKUs,
		Metadata: sdkschema.ResponseMetadata{
			Provider: ProviderStorageName,
			Resource: v1.StorageSKUResource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}
