package rest

import (
	"fmt"
	"net/http"
	"strconv"

	sdkauth "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
)

const (
	// RoleAPIVersion is the API version string used in response metadata.
	RoleAPIVersion = roledom.Version
	// RoleResource is the resource name.
	RoleResource = roledom.Resource
)

// RoleIdentity carries identity for a single role resource.
type RoleIdentity struct {
	name            string
	tenant          string
	resourceVersion string
}

func (ri *RoleIdentity) GetName() string      { return ri.name }
func (ri *RoleIdentity) GetVersion() string   { return ri.resourceVersion }
func (ri *RoleIdentity) GetTenant() string    { return ri.tenant }
func (ri *RoleIdentity) GetWorkspace() string { return "" }

var _ persistence.IdentifiableResource = (*RoleIdentity)(nil)

// ListParamsFromAPI converts SDK ListRolesParams to resource.ListParams.
func ListParamsFromAPI(params sdkauth.ListRolesParams, tenant string) resource.ListParams {
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
			Tenant: tenant,
		},
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// RoleToAPIWithVerb returns a func that converts a Role to its SDK representation with the given verb.
func RoleToAPIWithVerb(verb string) func(r *roledom.Role) *sdkschema.Role {
	return func(r *roledom.Role) *sdkschema.Role {
		return roleToAPI(*r, verb)
	}
}

// RoleIteratorToAPI converts a list of Role to an SDK RoleIterator.
func RoleIteratorToAPI(roles []*roledom.Role, nextSkipToken *string) *sdkauth.RoleIterator {
	items := make([]sdkschema.Role, len(roles))
	for i, r := range roles {
		items[i] = *(roleToAPI(*r, http.MethodGet))
	}

	iterator := &sdkauth.RoleIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: roledom.ProviderID,
			Resource: RoleResource,
			Verb:     http.MethodGet,
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// roleToAPI converts a Role to a schema.Role with the given verb.
func roleToAPI(r roledom.Role, verb string) *sdkschema.Role {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(r.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	sdk := &sdkschema.Role{
		Metadata: &sdkschema.GlobalTenantResourceMetadata{
			ApiVersion:      RoleAPIVersion,
			CreatedAt:       r.CreatedAt,
			LastModifiedAt:  r.UpdatedAt,
			Kind:            sdkschema.GlobalTenantResourceMetadataKindResourceKindRole,
			Name:            r.Name,
			Tenant:          r.Tenant,
			Provider:        r.Provider,
			Resource:        fmt.Sprintf(commondomain.ResourceFormat, sdkschema.GlobalTenantResourceMetadataKindResourceKindRole, r.Name),
			Ref:             fmt.Sprintf(r.Provider+"/"+commondomain.TenantScopedResourceFormat, r.Tenant, sdkschema.GlobalTenantResourceMetadataKindResourceKindRole, r.Name),
			ResourceVersion: resourceVersion,
			Verb:            verb,
		},
		Labels:      r.Labels,
		Annotations: r.Annotations,
		Extensions:  r.Extensions,
		Spec:        roleSpecToAPI(r.Spec),
	}
	if sdk.Labels == nil {
		sdk.Labels = make(sdkschema.Labels)
	}
	if r.Status != nil {
		sdk.Status = &sdkschema.RoleStatus{
			State:      commonfrontend.ResourceStateToAPI(r.Status.State),
			Conditions: commonfrontend.ConditionsToAPI(r.Status.Conditions),
		}
	}
	if r.DeletedAt != nil {
		sdk.Metadata.DeletedAt = r.DeletedAt
	}
	return sdk
}

// RoleFromAPI converts an SDK Role to a domain Role.
func RoleFromAPI(api sdkschema.Role, id *RoleIdentity) *roledom.Role {
	r := &roledom.Role{
		Spec: roleSpecFromAPI(api.Spec),
	}
	r.Name = id.GetName()
	r.ResourceVersion = id.GetVersion()
	r.Provider = roledom.ProviderID
	r.Tenant = id.GetTenant()
	r.Labels = api.Labels
	r.Annotations = api.Annotations
	r.Extensions = api.Extensions

	return r
}

// roleSpecToAPI converts a domain RoleSpec to an SDK RoleSpec.
func roleSpecToAPI(spec roledom.RoleSpec) sdkschema.RoleSpec {
	permissions := make([]sdkschema.Permission, len(spec.Permissions))
	for i, p := range spec.Permissions {
		resources := make([]string, len(p.Resources))
		copy(resources, p.Resources)
		verbs := make([]string, len(p.Verb))
		copy(verbs, p.Verb)
		permissions[i] = sdkschema.Permission{
			Provider:  p.Provider,
			Resources: resources,
			Verb:      verbs,
		}
	}
	return sdkschema.RoleSpec{Permissions: permissions}
}

// roleSpecFromAPI converts an SDK RoleSpec to a domain RoleSpec.
func roleSpecFromAPI(spec sdkschema.RoleSpec) roledom.RoleSpec {
	permissions := make([]roledom.Permission, len(spec.Permissions))
	for i, p := range spec.Permissions {
		resources := make([]string, len(p.Resources))
		copy(resources, p.Resources)
		verbs := make([]string, len(p.Verb))
		copy(verbs, p.Verb)
		permissions[i] = roledom.Permission{
			Provider:  p.Provider,
			Resources: resources,
			Verb:      verbs,
		}
	}
	return roledom.RoleSpec{Permissions: permissions}
}
