package storage

import (
	"fmt"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/regional/storage/block-storages/v1"
	v1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/regional/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/status"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/config"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional/consts"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

func DomainToAPIWithVerb(verb string) func(domain *regional.BlockStorageDomain) *sdkschema.BlockStorage {
	return func(domain *regional.BlockStorageDomain) *sdkschema.BlockStorage {
		sdk := BlockStorageToAPI(domain)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// BlockStorageToAPI converts a BlockStorageDomain to its SDK representation.
func BlockStorageToAPI(domain *regional.BlockStorageDomain) *sdkschema.BlockStorage {
	resVersion := int64(0)
	// resourceVersion is best-effort numeric
	if rv, err := strconv.ParseInt(domain.ResourceVersion, 10, 64); err == nil {
		resVersion = rv
	}

	bs := &sdkschema.BlockStorage{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:     v1.Version,
			CreatedAt:      domain.CreatedAt,
			LastModifiedAt: domain.UpdatedAt,
			Kind:           sdkschema.RegionalWorkspaceResourceMetadataKind(sdkschema.RegionalResourceMetadataKindResourceKindBlockStorage),
			Name:           domain.Name,
			Tenant:         domain.Tenant,
			Workspace:      domain.Workspace,
			Provider:       domain.Provider,
			Region:         domain.Region,
			Resource: fmt.Sprintf(
				regional.WorkspaceScopedResourceFormat,
				domain.Tenant,
				domain.Workspace,
				schema.RegionalResourceMetadataKindResourceKindBlockStorage,
				domain.Name,
			),
			Ref:             fmt.Sprintf(regional.ResourceFormat, sdkschema.RegionalResourceMetadataKindResourceKindBlockStorage, domain.Name),
			ResourceVersion: resVersion,
		},
		Labels:      domain.Labels,
		Annotations: domain.Annotations,
		Extensions:  domain.Extensions,
		Spec: sdkschema.BlockStorageSpec{
			SizeGB: domain.Spec.SizeGB,
			SkuRef: referenceObjectToAPI(domain.Spec.SkuRef),
		},
	}

	// TODO: better solution to replace this workaround
	if bs.Labels == nil {
		bs.Labels = make(sdkschema.Labels)
	}

	if domain.Spec.SourceImageRef != nil {
		bs.Spec.SourceImageRef = referenceObjectPtrToAPI(domain.Spec.SourceImageRef)
	}

	if domain.Status != nil {
		bs.Status = &sdkschema.BlockStorageStatus{
			SizeGB:     domain.Status.SizeGB,
			Conditions: status.MapConditionDomainsToAPI(domain.Status.Conditions),
		}
		if domain.Status.AttachedTo != nil {
			bs.Status.AttachedTo = referenceObjectPtrToAPI(domain.Status.AttachedTo)
		}

		bs.Status.State = sdkschema.ResourceState(domain.Status.State)
	}
	if domain.DeletedAt != nil {
		bs.Metadata.DeletedAt = domain.DeletedAt
	}
	return bs
}

// BlockStorageListParamsFromAPI converts SDK ListBlockStoragesParams to model.ListParams.
func BlockStorageListParamsFromAPI(params sdkstorage.ListBlockStoragesParams, tenant, workspace string) model.ListParams {
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
			Tenant:    tenant,
			Workspace: workspace,
		},
	}
}

// BlockStorageDomainToAPIIterator converts a list of BlockStorageDomain to an SDK BlockStorageIterator.
func BlockStorageDomainToAPIIterator(domains []*regional.BlockStorageDomain, nextSkipToken *string) *sdkstorage.BlockStorageIterator {
	items := make([]sdkschema.BlockStorage, len(domains))
	for i := range domains {
		mapped := BlockStorageToAPI(domains[i])
		items[i] = *mapped
	}

	iterator := &sdkstorage.BlockStorageIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: consts.StorageProvider,
			Resource: blockstoragev1.BlockStorageResource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// BlockStorageFromAPI converts an SDK BlockStorage to a BlockStorageDomain.
func BlockStorageFromAPI(sdk sdkschema.BlockStorage, params port.IdentifiableResource) *regional.BlockStorageDomain {
	domain := &regional.BlockStorageDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name:            params.GetName(),
				ResourceVersion: params.GetVersion(),
				Provider:        consts.StorageProvider,
			},
			Scope: scope.Scope{
				Tenant:    params.GetTenant(),
				Workspace: params.GetWorkspace(),
			},
			Region:      config.Singleton().Region(),
			Labels:      sdk.Labels,
			Annotations: sdk.Annotations,
			Extensions:  sdk.Extensions,
		},
		Spec: regional.BlockStorageSpecDomain{
			SizeGB: sdk.Spec.SizeGB,
			SkuRef: referenceObjectFromAPI(sdk.Spec.SkuRef),
		},
	}

	if sdk.Spec.SourceImageRef != nil {
		domain.Spec.SourceImageRef = new(referenceObjectFromAPI(*sdk.Spec.SourceImageRef))
	}

	return domain
}

// Helper functions for reference object conversion

func referenceObjectToAPI(ref regional.ReferenceObjectDomain) sdkschema.Reference {
	return sdkschema.Reference{
		Provider:  ref.Provider,
		Region:    ref.Region,
		Resource:  ref.Resource,
		Tenant:    ref.Tenant,
		Workspace: ref.Workspace,
	}
}

func toPtrOrNil[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

func referenceObjectPtrToAPI(ref *regional.ReferenceObjectDomain) *sdkschema.Reference {
	if ref == nil {
		return nil
	}
	r := referenceObjectToAPI(*ref)
	return &r
}

func referenceObjectFromAPI(ref sdkschema.Reference) regional.ReferenceObjectDomain {
	return regional.ReferenceObjectDomain{
		Provider:  ref.Provider,
		Region:    ref.Region,
		Resource:  ref.Resource,
		Tenant:    ref.Tenant,
		Workspace: ref.Workspace,
	}
}
