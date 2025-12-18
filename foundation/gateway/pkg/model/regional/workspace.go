package regional

import (
	"fmt"
	"strconv"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// NOTE: Should base URLs and Provider names be passed at API deployment time?
// 	Base URLs definitely should, provider names could be retrieved from cluster (not sure if it's worth the effort).
const (
	WorkspaceBaseURL      = "/providers/seca.workspace"
	ProviderWorkspaceName = "seca.workspace/v1"
)

type WorkspaceDomain struct {
	Metadata

	Spec   WorkspaceSpec
	Status WorkspaceStatusDomain
}

type WorkspaceSpec = map[string]interface{}

type WorkspaceStatusDomain struct {
	StatusDomain
	ResourceCount *int
}

// MapWorkspaceDomainToAPI maps a WorkspaceDomain to schema.Workspace API object.
func MapWorkspaceDomainToAPI(domain WorkspaceDomain, verb string) schema.Workspace {
	resVersion := 0
	// resourceVersion is best-effort numeric
	if rv, err := strconv.Atoi(domain.ResourceVersion); err == nil {
		resVersion = rv
	}

	refObj := schema.ReferenceObject{
		Resource: fmt.Sprintf(ResourceFormat, schema.RegionalResourceMetadataKindResourceKindWorkspace, domain.Name),
		Provider: &domain.Provider,
		Region:   &domain.Region,
		Tenant:   &domain.Tenant,
	}
	ref := schema.Reference{}
	_ = ref.FromReferenceObject(refObj) // ignore mapping error, not critical internally

	var resourceState *schema.ResourceState
	if domain.Status.State != nil {
		rs := mapResourceStateDomainToAPI(*domain.Status.State)
		resourceState = &rs
	}
	sdk := schema.Workspace{
		Spec: domain.Spec,
		Metadata: &schema.RegionalResourceMetadata{
			ApiVersion:      regionsv1.Version,
			CreatedAt:       domain.CreatedAt,
			LastModifiedAt:  domain.UpdatedAt,
			Kind:            schema.RegionalResourceMetadataKindResourceKindWorkspace,
			Name:            domain.Name,
			Provider:        domain.Provider,
			Region:          domain.Region,
			Resource:        fmt.Sprintf(TenantScopedResourceFormat, domain.Tenant, schema.RegionalResourceMetadataKindResourceKindWorkspace, domain.Name),
			Ref:             &ref,
			ResourceVersion: resVersion,
			Verb:            verb,
		},
		Labels:      domain.Labels,
		Annotations: domain.Annotations,
		Extensions:  domain.Extensions,
		Status: &schema.WorkspaceStatus{
			ResourceCount: domain.Status.ResourceCount,
			State:         resourceState,
			Conditions:    mapConditionsInStatusDomainToAPI(domain.Status.StatusDomain),
		},
	}
	if domain.DeletedAt != nil {
		sdk.Metadata.DeletedAt = domain.DeletedAt
	}
	return sdk
}

// MapWorkspaceAPIToDomain maps a schema.Workspace API object to WorkspaceDomain.
func MapWorkspaceAPIToDomain(sdk schema.Workspace, params UpsertParams) WorkspaceDomain {
	return WorkspaceDomain{
		Metadata: Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: params.Name,
			},
			Scope: scope.Scope{
				Tenant: params.GetTenant(),
			},
			Annotations: sdk.Annotations,
			Labels:      sdk.Labels,
			Extensions:  sdk.Extensions,
		},
		Spec: sdk.Spec,
	}
}
