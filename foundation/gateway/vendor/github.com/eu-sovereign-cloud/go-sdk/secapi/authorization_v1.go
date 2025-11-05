package secapi

import (
	"context"
	"net/http"

	authorization "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"k8s.io/utils/ptr"
)

type AuthorizationV1 struct {
	API
	authorization authorization.ClientWithResponsesInterface
}

// Role

func (api *AuthorizationV1) ListRoles(ctx context.Context, tid TenantID) (*Iterator[schema.Role], error) {
	iter := Iterator[schema.Role]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.Role, *string, error) {
			resp, err := api.authorization.ListRolesWithResponse(ctx, schema.TenantPathParam(tid), &authorization.ListRolesParams{
				Accept: ptr.To(authorization.ListRolesParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *AuthorizationV1) GetRole(ctx context.Context, tref TenantReference) (*schema.Role, error) {
	if err := tref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.authorization.GetRoleWithResponse(ctx, schema.TenantPathParam(tref.Tenant), string(tref.Name), api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *AuthorizationV1) CreateOrUpdateRoleWithParams(ctx context.Context, role *schema.Role, params *authorization.CreateOrUpdateRoleParams) (*schema.Role, error) {
	if err := api.validateMetadata(role.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.authorization.CreateOrUpdateRoleWithResponse(ctx, schema.TenantPathParam(role.Metadata.Tenant), string(role.Metadata.Name), params, *role, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *AuthorizationV1) CreateOrUpdateRole(ctx context.Context, role *schema.Role) (*schema.Role, error) {
	return api.CreateOrUpdateRoleWithParams(ctx, role, nil)
}

func (api *AuthorizationV1) DeleteRoleWithParams(ctx context.Context, role *schema.Role, params *authorization.DeleteRoleParams) error {
	if err := api.validateMetadata(role.Metadata); err != nil {
		return err
	}

	resp, err := api.authorization.DeleteRoleWithResponse(ctx, role.Metadata.Tenant, role.Metadata.Name, params, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *AuthorizationV1) DeleteRole(ctx context.Context, role *schema.Role) error {
	return api.DeleteRoleWithParams(ctx, role, nil)
}

// Role Assignment

func (api *AuthorizationV1) ListRoleAssignments(ctx context.Context, tid TenantID) (*Iterator[schema.RoleAssignment], error) {
	iter := Iterator[schema.RoleAssignment]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.RoleAssignment, *string, error) {
			resp, err := api.authorization.ListRoleAssignmentsWithResponse(ctx, schema.TenantPathParam(tid), &authorization.ListRoleAssignmentsParams{
				Accept: ptr.To(authorization.ListRoleAssignmentsParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *AuthorizationV1) GetRoleAssignment(ctx context.Context, tref TenantReference) (*schema.RoleAssignment, error) {
	if err := tref.validate(); err != nil {
		return nil, err
	}

	resp, err := api.authorization.GetRoleAssignmentWithResponse(ctx, schema.TenantPathParam(tref.Tenant), string(tref.Name), api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func (api *AuthorizationV1) CreateOrUpdateRoleAssignmentWithParams(ctx context.Context, assign *schema.RoleAssignment, params *authorization.CreateOrUpdateRoleAssignmentParams) (*schema.RoleAssignment, error) {
	if err := api.validateMetadata(assign.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.authorization.CreateOrUpdateRoleAssignmentWithResponse(ctx, schema.TenantPathParam(assign.Metadata.Tenant), string(assign.Metadata.Name), params, *assign, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if err = checkSuccessPutStatusCodes(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.JSON200, nil
	} else {
		return resp.JSON201, nil
	}
}

func (api *AuthorizationV1) CreateOrUpdateRoleAssignment(ctx context.Context, assign *schema.RoleAssignment) (*schema.RoleAssignment, error) {
	return api.CreateOrUpdateRoleAssignmentWithParams(ctx, assign, nil)
}

func (api *AuthorizationV1) DeleteRoleAssignmentWithParams(ctx context.Context, assign *schema.RoleAssignment, params *authorization.DeleteRoleAssignmentParams) error {
	if err := api.validateMetadata(assign.Metadata); err != nil {
		return err
	}

	resp, err := api.authorization.DeleteRoleAssignmentWithResponse(ctx, assign.Metadata.Tenant, assign.Metadata.Name, params, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

func (api *AuthorizationV1) DeleteRoleAssignment(ctx context.Context, assign *schema.RoleAssignment) error {
	return api.DeleteRoleAssignmentWithParams(ctx, assign, nil)
}

func (api *AuthorizationV1) validateMetadata(metadata *schema.GlobalResourceMetadata) error {
	if metadata == nil {
		return ErrNoMetatada
	}

	if metadata.Tenant == "" {
		return ErrNoMetatadaTenant
	}

	return nil
}

func newAuthorizationV1(client *GlobalClient, authorizationsUrl string) (*AuthorizationV1, error) {
	authorization, err := authorization.NewClientWithResponses(authorizationsUrl)
	if err != nil {
		return nil, err
	}

	return &AuthorizationV1{API: API{authToken: client.authToken}, authorization: authorization}, nil
}
