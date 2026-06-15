package storage

import (
	"fmt"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	imagev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/storage/images/v1"
	v1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/rest2domain/status"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/rest2domain/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/reference"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/config"

	model "github.com/eu-sovereign-cloud/ecp/foundation/models"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional/consts"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

func ImageDomainToAPIWithVerb(verb string) func(domain *regional.ImageDomain) *sdkschema.Image {
	return func(domain *regional.ImageDomain) *sdkschema.Image {
		sdk := imageDomainToAPI(domain)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// imageDomainToAPI converts an ImageDomain to its SDK representation.
func imageDomainToAPI(domain *regional.ImageDomain) *sdkschema.Image {
	resVersion := int64(0)
	// resourceVersion is best-effort numeric
	if rv, err := strconv.ParseInt(domain.ResourceVersion, 10, 64); err == nil {
		resVersion = rv
	}

	img := &sdkschema.Image{
		Metadata: &sdkschema.RegionalResourceMetadata{
			ApiVersion:     v1.Version,
			CreatedAt:      domain.CreatedAt,
			LastModifiedAt: domain.UpdatedAt,
			Kind:           sdkschema.RegionalResourceMetadataKindResourceKindImage,
			Name:           domain.Name,
			Tenant:         domain.Tenant,
			Provider:       domain.Provider,
			Region:         domain.Region,
			Resource:       fmt.Sprintf(regional.ResourceFormat, sdkschema.RegionalResourceMetadataKindResourceKindImage, domain.Name),
			Ref: fmt.Sprintf(
				domain.Provider+"/"+regional.TenantScopedResourceFormat,
				domain.Tenant,
				sdkschema.RegionalResourceMetadataKindResourceKindImage,
				domain.Name,
			),
			ResourceVersion: resVersion,
		},
		Labels:      domain.Labels,
		Annotations: domain.Annotations,
		Extensions:  domain.Extensions,
		Spec: sdkschema.ImageSpec{
			BlockStorageRef: reference.ToAPI(domain.Spec.BlockStorageRef),
			CpuArchitecture: sdkschema.ImageSpecCpuArchitecture(domain.Spec.CpuArchitecture),
			Initializer:     sdkschema.ImageSpecInitializer(domain.Spec.Initializer),
			Boot:            sdkschema.ImageSpecBoot(domain.Spec.Boot),
		},
	}

	// TODO: better solution to replace this workaround
	if img.Labels == nil {
		img.Labels = make(sdkschema.Labels)
	}

	if domain.Status != nil {
		img.Status = &sdkschema.ImageStatus{
			SizeMB:     domain.Status.SizeMB,
			Conditions: status.ConditionDomainsToAPI(domain.Status.Conditions),
			State:      sdkschema.ResourceState(domain.Status.State),
		}
	}
	if domain.DeletedAt != nil {
		img.Metadata.DeletedAt = domain.DeletedAt
	}
	return img
}

// ImageListParamsFromAPI converts SDK ListImagesParams to model.ListParams.
func ImageListParamsFromAPI(params sdkstorage.ListImagesParams, tenant string) model.ListParams {
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
		Scope: scope.Scope{
			Tenant: tenant,
		},
	}
}

// ImageDomainToAPIIterator converts a list of ImageDomain to an SDK ImageIterator.
func ImageDomainToAPIIterator(domains []*regional.ImageDomain, nextSkipToken *string) *sdkstorage.ImageIterator {
	items := make([]sdkschema.Image, len(domains))
	for i := range domains {
		mapped := imageDomainToAPI(domains[i])
		items[i] = *mapped
	}

	iterator := &sdkstorage.ImageIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: consts.StorageProvider,
			Resource: imagev1.ImageResource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// APIToImageDomain converts an SDK Image to an ImageDomain.
func APIToImageDomain(sdk sdkschema.Image, params port.IdentifiableResource) *regional.ImageDomain {
	return &regional.ImageDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name:            params.GetName(),
				ResourceVersion: params.GetVersion(),
				Provider:        consts.StorageProvider,
			},
			Scope: scope.Scope{
				Tenant: params.GetTenant(),
			},
			Region:      config.Singleton().Region(),
			Labels:      sdk.Labels,
			Annotations: sdk.Annotations,
			Extensions:  sdk.Extensions,
		},
		Spec: regional.ImageSpecDomain{
			BlockStorageRef: reference.FromAPI(sdk.Spec.BlockStorageRef),
			CpuArchitecture: string(sdk.Spec.CpuArchitecture),
			Initializer:     string(sdk.Spec.Initializer),
			Boot:            string(sdk.Spec.Boot),
		},
	}
}
