// Package rest provides REST↔domain conversion and HTTP handlers for the block storage resource.
package rest

import (
	"fmt"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resources/common/frontend"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
)

const (
	// BlockStorageAPIVersion is the API version string used in response metadata.
	BlockStorageAPIVersion = bsdom.Version
	// BlockStorageResource is the resource name.
	BlockStorageResource = bsdom.Resource
	// ResourceFormat formats a resource path string.
	ResourceFormat = "%s/%s"
	// WorkspaceScopedResourceFormat formats a workspace-scoped resource ref.
	WorkspaceScopedResourceFormat = "tenants/%s/workspaces/%s/providers/%s/%s"
)

// BlockStorageIdentity carries identity for a single block-storage resource.
type BlockStorageIdentity struct {
	name            string
	tenant          string
	workspace       string
	resourceVersion string
}

func (b *BlockStorageIdentity) GetName() string      { return b.name }
func (b *BlockStorageIdentity) GetVersion() string   { return b.resourceVersion }
func (b *BlockStorageIdentity) GetTenant() string    { return b.tenant }
func (b *BlockStorageIdentity) GetWorkspace() string { return b.workspace }

var _ persistence.IdentifiableResource = (*BlockStorageIdentity)(nil)

// ListParamsFromAPI converts SDK ListBlockStoragesParams to resource.ListParams.
func ListParamsFromAPI(params sdkstorage.ListBlockStoragesParams, tenant, workspace string) resource.ListParams {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	return resource.ListParams{
		Scope: resource.Scope{
			Tenant:    tenant,
			Workspace: workspace,
		},
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// BlockStorageDomainToAPIWithVerb returns a func that converts a BlockStorage to its SDK representation with the given verb.
func BlockStorageDomainToAPIWithVerb(verb string) func(domain *bsdom.BlockStorage) *sdkschema.BlockStorage {
	return func(domain *bsdom.BlockStorage) *sdkschema.BlockStorage {
		sdk := blockStorageDomainToAPI(domain)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// blockStorageDomainToAPI converts a BlockStorage to its SDK representation.
func blockStorageDomainToAPI(domain *bsdom.BlockStorage) *sdkschema.BlockStorage {
	resVersion := int64(0)
	if rv, err := strconv.ParseInt(domain.ResourceVersion, 10, 64); err == nil {
		resVersion = rv
	}

	bs := &sdkschema.BlockStorage{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:     BlockStorageAPIVersion,
			CreatedAt:      domain.CreatedAt,
			LastModifiedAt: domain.UpdatedAt,
			Kind:           sdkschema.RegionalWorkspaceResourceMetadataKind(sdkschema.RegionalResourceMetadataKindResourceKindBlockStorage),
			Name:           domain.Name,
			Tenant:         domain.Tenant,
			Workspace:      domain.Workspace,
			Provider:       domain.Provider,
			Region:         domain.Region,
			Resource:       fmt.Sprintf(ResourceFormat, sdkschema.RegionalResourceMetadataKindResourceKindBlockStorage, domain.Name),
			Ref: fmt.Sprintf(
				domain.Provider+"/"+WorkspaceScopedResourceFormat,
				domain.Tenant,
				domain.Workspace,
				sdkschema.RegionalResourceMetadataKindResourceKindBlockStorage,
				domain.Name,
			),
			ResourceVersion: resVersion,
		},
		Labels:      domain.Labels,
		Annotations: domain.Annotations,
		Extensions:  domain.Extensions,
		Spec: sdkschema.BlockStorageSpec{
			SizeGB: domain.Spec.SizeGB,
			SkuRef: commonfrontend.ToAPI(domain.Spec.SkuRef),
		},
	}

	if bs.Labels == nil {
		bs.Labels = make(sdkschema.Labels)
	}

	if domain.Spec.SourceImageRef != nil {
		bs.Spec.SourceImageRef = commonfrontend.PtrToAPI(domain.Spec.SourceImageRef)
	}

	if domain.Status != nil {
		bs.Status = &sdkschema.BlockStorageStatus{
			SizeGB:     domain.Status.SizeGB,
			Conditions: commonfrontend.ConditionDomainsToAPI(domain.Status.Conditions),
			State:      commonfrontend.ResourceStateDomainToAPI(domain.Status.State),
		}
		if domain.Status.AttachedTo != nil {
			bs.Status.AttachedTo = commonfrontend.PtrToAPI(domain.Status.AttachedTo)
		}
	}
	if domain.DeletedAt != nil {
		bs.Metadata.DeletedAt = domain.DeletedAt
	}
	return bs
}

// BlockStorageDomainToAPIIterator converts a list of BlockStorage to an SDK BlockStorageIterator.
func BlockStorageDomainToAPIIterator(domains []*bsdom.BlockStorage, nextSkipToken *string) *sdkstorage.BlockStorageIterator {
	items := make([]sdkschema.BlockStorage, len(domains))
	for i := range domains {
		items[i] = *blockStorageDomainToAPI(domains[i])
	}

	iterator := &sdkstorage.BlockStorageIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: bsdom.ProviderID,
			Resource: BlockStorageResource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// APIToBlockStorageDomain converts an SDK BlockStorage to a BlockStorage.
func APIToBlockStorageDomain(sdk sdkschema.BlockStorage, id *BlockStorageIdentity, region string) *bsdom.BlockStorage {
	domain := &bsdom.BlockStorage{
		Spec: bsdom.BlockStorageSpec{
			SizeGB: sdk.Spec.SizeGB,
			SkuRef: commonfrontend.FromAPI(sdk.Spec.SkuRef),
		},
	}
	domain.Name = id.GetName()
	domain.ResourceVersion = id.GetVersion()
	domain.Provider = bsdom.ProviderID
	domain.Tenant = id.GetTenant()
	domain.Workspace = id.GetWorkspace()
	domain.Region = region
	domain.Labels = sdk.Labels
	domain.Annotations = sdk.Annotations
	domain.Extensions = sdk.Extensions

	if sdk.Spec.SourceImageRef != nil {
		ref := commonfrontend.FromAPI(*sdk.Spec.SourceImageRef)
		domain.Spec.SourceImageRef = &ref
	}

	return domain
}
