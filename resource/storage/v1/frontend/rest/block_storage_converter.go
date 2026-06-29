package rest

import (
	"fmt"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

const (
	// BlockStorageAPIVersion is the API version string used in response metadata.
	BlockStorageAPIVersion = bsdom.Version
	// BlockStorageResource is the resource name.
	BlockStorageResource = bsdom.Resource
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

// blockStorageListParamsFromAPI converts SDK ListBlockStoragesParams to resource.ListParams.
func blockStorageListParamsFromAPI(params sdkstorage.ListBlockStoragesParams, tenant, workspace string) resource.ListParams {
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

// blockStorageToAPIWithVerb returns a func that converts a BlockStorage to its SDK representation with the given verb.
func blockStorageToAPIWithVerb(verb string) func(bs *bsdom.BlockStorage) *sdkschema.BlockStorage {
	return func(bs *bsdom.BlockStorage) *sdkschema.BlockStorage {
		sdk := blockStorageToAPI(bs)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// blockStorageToAPI converts a BlockStorage to its SDK representation.
func blockStorageToAPI(bs *bsdom.BlockStorage) *sdkschema.BlockStorage {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(bs.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	out := &sdkschema.BlockStorage{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:     BlockStorageAPIVersion,
			CreatedAt:      bs.CreatedAt,
			LastModifiedAt: bs.UpdatedAt,
			Kind:           sdkschema.RegionalWorkspaceResourceMetadataKind(sdkschema.RegionalResourceMetadataKindResourceKindBlockStorage),
			Name:           bs.Name,
			Tenant:         bs.Tenant,
			Workspace:      bs.Workspace,
			Provider:       bs.Provider,
			Region:         bs.Region,
			Resource:       fmt.Sprintf(resourceFormat, sdkschema.RegionalResourceMetadataKindResourceKindBlockStorage, bs.Name),
			Ref: fmt.Sprintf(
				bs.Provider+"/"+workspaceScopedResourceFormat,
				bs.Tenant,
				bs.Workspace,
				sdkschema.RegionalResourceMetadataKindResourceKindBlockStorage,
				bs.Name,
			),
			ResourceVersion: resourceVersion,
		},
		Labels:      bs.Labels,
		Annotations: bs.Annotations,
		Extensions:  bs.Extensions,
		Spec: sdkschema.BlockStorageSpec{
			SizeGB: bs.Spec.SizeGB,
			SkuRef: commonfrontend.ReferenceToAPI(bs.Spec.SkuRef),
		},
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}

	if bs.Spec.SourceImageRef != nil {
		out.Spec.SourceImageRef = commonfrontend.ReferencePtrToAPI(bs.Spec.SourceImageRef)
	}

	if bs.Status != nil {
		out.Status = &sdkschema.BlockStorageStatus{
			SizeGB:     bs.Status.SizeGB,
			Conditions: commonfrontend.ConditionsToAPI(bs.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(bs.Status.State),
		}
		if bs.Status.AttachedTo != nil {
			out.Status.AttachedTo = commonfrontend.ReferencePtrToAPI(bs.Status.AttachedTo)
		}
	}
	if bs.DeletedAt != nil {
		out.Metadata.DeletedAt = bs.DeletedAt
	}
	return out
}

// blockStorageIteratorToAPI converts a list of BlockStorage to an SDK BlockStorageIterator.
func blockStorageIteratorToAPI(bss []*bsdom.BlockStorage, nextSkipToken *string) *sdkstorage.BlockStorageIterator {
	items := make([]sdkschema.BlockStorage, len(bss))
	for i := range bss {
		items[i] = *blockStorageToAPI(bss[i])
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

// blockStorageFromAPI converts an SDK BlockStorage to a BlockStorage.
func blockStorageFromAPI(sdk sdkschema.BlockStorage, id *BlockStorageIdentity, region string) *bsdom.BlockStorage {
	bs := &bsdom.BlockStorage{
		Spec: bsdom.BlockStorageSpec{
			SizeGB: sdk.Spec.SizeGB,
			SkuRef: commonfrontend.ReferenceFromAPI(sdk.Spec.SkuRef),
		},
	}
	bs.Name = id.GetName()
	bs.ResourceVersion = id.GetVersion()
	bs.Provider = bsdom.ProviderID
	bs.Tenant = id.GetTenant()
	bs.Workspace = id.GetWorkspace()
	bs.Region = region
	bs.Labels = sdk.Labels
	bs.Annotations = sdk.Annotations
	bs.Extensions = sdk.Extensions

	if sdk.Spec.SourceImageRef != nil {
		ref := commonfrontend.ReferenceFromAPI(*sdk.Spec.SourceImageRef)
		bs.Spec.SourceImageRef = &ref
	}

	return bs
}
