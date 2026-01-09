package storage

import (
	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"k8s.io/utils/ptr"

	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/block-storages/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

// BlockStorageToAPI converts a BlockStorageDomain to its SDK representation.
func BlockStorageToAPI(domain *regional.BlockStorageDomain) *sdkschema.BlockStorage {
	bs := &sdkschema.BlockStorage{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			Name:      domain.Name,
			Tenant:    domain.GetTenant(),
			Workspace: domain.GetWorkspace(),
		},
		Spec: sdkschema.BlockStorageSpec{
			SizeGB: domain.Spec.SizeGB,
			SkuRef: referenceObjectToAPI(domain.Spec.SkuRef),
		},
		Labels:      domain.Labels,
		Annotations: domain.Annotations,
		Extensions:  domain.Extensions,
	}

	if domain.Spec.SourceImageRef != nil {
		bs.Spec.SourceImageRef = referenceObjectPtrToAPI(domain.Spec.SourceImageRef)
	}

	if domain.Status != nil {
		bs.Status = &sdkschema.BlockStorageStatus{
			SizeGB:     domain.Status.SizeGB,
			Conditions: statusConditionsToAPI(domain.Status.Conditions),
		}
		if domain.Status.AttachedTo != nil {
			bs.Status.AttachedTo = referenceObjectPtrToAPI(domain.Status.AttachedTo)
		}
		if domain.Status.State != nil {
			state := sdkschema.ResourceState(*domain.Status.State)
			bs.Status.State = &state
		}
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
			Provider: ProviderStorageName,
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
func BlockStorageFromAPI(sdk sdkschema.BlockStorage, params regional.UpsertParams) *regional.BlockStorageDomain {
	domain := &regional.BlockStorageDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: params.GetName(),
			},
			Scope: scope.Scope{
				Tenant:    params.GetTenant(),
				Workspace: params.GetWorkspace(),
			},
			Labels:      sdk.Labels,
			Annotations: sdk.Annotations,
			Extensions:  sdk.Extensions,
		},
		Spec: regional.BlockStorageSpec{
			SizeGB: sdk.Spec.SizeGB,
			SkuRef: referenceObjectFromAPI(sdk.Spec.SkuRef),
		},
	}

	if sdk.Spec.SourceImageRef != nil {
		ref := referenceObjectFromAPI(*sdk.Spec.SourceImageRef)
		domain.Spec.SourceImageRef = &ref
	}

	return domain
}

// Helper functions for reference object conversion

func referenceObjectToAPI(ref regional.ReferenceObject) sdkschema.Reference {
	refObj := sdkschema.ReferenceObject{
		Provider:  ptr.To(ref.Provider),
		Region:    ptr.To(ref.Region),
		Resource:  ref.Resource,
		Tenant:    ptr.To(ref.Tenant),
		Workspace: ptr.To(ref.Workspace),
	}
	var result sdkschema.Reference
	_ = result.FromReferenceObject(refObj)
	return result
}

func referenceObjectPtrToAPI(ref *regional.ReferenceObject) *sdkschema.Reference {
	if ref == nil {
		return nil
	}
	r := referenceObjectToAPI(*ref)
	return &r
}

func referenceObjectFromAPI(ref sdkschema.Reference) regional.ReferenceObject {
	// Handle the Reference union type - try ReferenceObject first
	refObj, err := ref.AsReferenceObject()
	if err == nil {
		return regional.ReferenceObject{
			Provider:  ptr.Deref(refObj.Provider, ""),
			Region:    ptr.Deref(refObj.Region, ""),
			Resource:  refObj.Resource,
			Tenant:    ptr.Deref(refObj.Tenant, ""),
			Workspace: ptr.Deref(refObj.Workspace, ""),
		}
	}
	// Handle ReferenceURN as a fallback
	refURN, err := ref.AsReferenceURN()
	if err == nil {
		return regional.ReferenceObject{
			Resource: refURN,
		}
	}
	return regional.ReferenceObject{}
}

func statusConditionsToAPI(conditions []regional.StatusCondition) []sdkschema.StatusCondition {
	result := make([]sdkschema.StatusCondition, len(conditions))
	for i, c := range conditions {
		result[i] = sdkschema.StatusCondition{
			LastTransitionAt: c.LastTransitionAt,
			Message:          ptr.To(c.Message),
			Reason:           ptr.To(c.Reason),
			State:            sdkschema.ResourceState(c.State),
			Type:             ptr.To(c.Type),
		}
	}
	return result
}
