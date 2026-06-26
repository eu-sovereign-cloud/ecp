package rest

import (
	"fmt"
	"strconv"

	sdkauthz "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
)

const (
	// RoleAssignmentAPIVersion is the API version string used in response metadata.
	RoleAssignmentAPIVersion = radom.Version
	// RoleAssignmentResource is the resource name.
	RoleAssignmentResource = radom.Resource
)

// RoleAssignmentIdentity carries identity for a single role assignment resource.
type RoleAssignmentIdentity struct {
	name            string
	tenant          string
	resourceVersion string
}

func (r *RoleAssignmentIdentity) GetName() string      { return r.name }
func (r *RoleAssignmentIdentity) GetVersion() string   { return r.resourceVersion }
func (r *RoleAssignmentIdentity) GetTenant() string    { return r.tenant }
func (r *RoleAssignmentIdentity) GetWorkspace() string { return "" }

var _ persistence.IdentifiableResource = (*RoleAssignmentIdentity)(nil)

// RoleAssignmentToAPIWithVerb returns a func that converts a RoleAssignment to its SDK representation with the given verb.
func RoleAssignmentToAPIWithVerb(verb string) func(ra *radom.RoleAssignment) *sdkschema.RoleAssignment {
	return func(ra *radom.RoleAssignment) *sdkschema.RoleAssignment {
		sdk := roleAssignmentToAPI(ra)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// roleAssignmentToAPI converts a RoleAssignment to its SDK representation.
func roleAssignmentToAPI(ra *radom.RoleAssignment) *sdkschema.RoleAssignment {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(ra.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	out := &sdkschema.RoleAssignment{
		Metadata: &sdkschema.GlobalTenantResourceMetadata{
			ApiVersion:     RoleAssignmentAPIVersion,
			CreatedAt:      ra.CreatedAt,
			LastModifiedAt: ra.UpdatedAt,
			Kind:           sdkschema.GlobalTenantResourceMetadataKindResourceKindRoleAssignment,
			Name:           ra.Name,
			Tenant:         ra.Tenant,
			Provider:       ra.Provider,
			Resource:       fmt.Sprintf(resourceFormat, sdkschema.GlobalTenantResourceMetadataKindResourceKindRoleAssignment, ra.Name),
			Ref: fmt.Sprintf(
				ra.Provider+"/"+tenantScopedResourceFormat,
				ra.Tenant,
				sdkschema.GlobalTenantResourceMetadataKindResourceKindRoleAssignment,
				ra.Name,
			),
			ResourceVersion: resourceVersion,
		},
		Labels:      ra.Labels,
		Annotations: ra.Annotations,
		Extensions:  ra.Extensions,
		Spec: sdkschema.RoleAssignmentSpec{
			Subs:   ra.Spec.Subs,
			Scopes: scopesToAPI(ra.Spec.Scopes),
			Roles:  ra.Spec.Roles,
		},
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}

	if ra.Status != nil {
		out.Status = &sdkschema.RoleAssignmentStatus{
			Conditions: commonfrontend.ConditionsToAPI(ra.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(ra.Status.State),
		}
	}
	if ra.DeletedAt != nil {
		out.Metadata.DeletedAt = ra.DeletedAt
	}
	return out
}

// RoleAssignmentIteratorToAPI converts a list of RoleAssignment to an SDK RoleAssignmentIterator.
func RoleAssignmentIteratorToAPI(ras []*radom.RoleAssignment, nextSkipToken *string) *sdkauthz.RoleAssignmentIterator {
	items := make([]sdkschema.RoleAssignment, len(ras))
	for i := range ras {
		items[i] = *roleAssignmentToAPI(ras[i])
	}

	iterator := &sdkauthz.RoleAssignmentIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: radom.ProviderID,
			Resource: RoleAssignmentResource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// RoleAssignmentFromAPI converts an SDK RoleAssignment to a RoleAssignment.
func RoleAssignmentFromAPI(sdk sdkschema.RoleAssignment, id *RoleAssignmentIdentity, region string) *radom.RoleAssignment {
	ra := &radom.RoleAssignment{
		Spec: radom.RoleAssignmentSpec{
			Subs:   sdk.Spec.Subs,
			Scopes: scopesFromAPI(sdk.Spec.Scopes),
			Roles:  sdk.Spec.Roles,
		},
	}
	ra.Name = id.GetName()
	ra.ResourceVersion = id.GetVersion()
	ra.Provider = radom.ProviderID
	ra.Tenant = id.GetTenant()
	ra.Region = region
	ra.Labels = sdk.Labels
	ra.Annotations = sdk.Annotations
	ra.Extensions = sdk.Extensions

	return ra
}

// scopesToAPI converts domain role assignment scopes into their SDK representation.
func scopesToAPI(scopes []radom.RoleAssignmentScope) []sdkschema.RoleAssignmentScope {
	if scopes == nil {
		return nil
	}
	out := make([]sdkschema.RoleAssignmentScope, len(scopes))
	for i, s := range scopes {
		out[i] = sdkschema.RoleAssignmentScope{
			Tenants:    s.Tenants,
			Regions:    s.Regions,
			Workspaces: s.Workspaces,
		}
	}
	return out
}

// scopesFromAPI converts SDK role assignment scopes into their domain representation.
func scopesFromAPI(scopes []sdkschema.RoleAssignmentScope) []radom.RoleAssignmentScope {
	if scopes == nil {
		return nil
	}
	out := make([]radom.RoleAssignmentScope, len(scopes))
	for i, s := range scopes {
		out[i] = radom.RoleAssignmentScope{
			Tenants:    s.Tenants,
			Regions:    s.Regions,
			Workspaces: s.Workspaces,
		}
	}
	return out
}
