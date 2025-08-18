package secapi

import (
	"context"
	"net/http"

	authorization "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"

	"k8s.io/utils/ptr"
)

type AuthorizationV1 struct {
	API
	authorization authorization.ClientWithResponsesInterface
}

// Role

func (api *AuthorizationV1) ListRoles(ctx context.Context, tid TenantID) (*Iterator[authorization.Role], error) {
	iter := Iterator[authorization.Role]{
		fn: func(ctx context.Context, skipToken *string) ([]authorization.Role, *string, error) {
			resp, err := api.authorization.ListRolesWithResponse(ctx, authorization.Tenant(tid), &authorization.ListRolesParams{
				Accept: ptr.To(authorization.Applicationjson),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *AuthorizationV1) GetRole(ctx context.Context, tref TenantReference) (*authorization.Role, error) {
	if err := validateTenantReference(tref); err != nil {
		return nil, err
	}

	resp, err := api.authorization.GetRoleWithResponse(ctx, authorization.Tenant(tref.Tenant), string(tref.Name), api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

func (api *AuthorizationV1) CreateOrUpdateRole(ctx context.Context, role *authorization.Role) (*authorization.Role, error) {
	if err := validateAuthorizationMetadataV1(role.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.authorization.CreateOrUpdateRoleWithResponse(ctx, role.Metadata.Tenant, role.Metadata.Name,
		&authorization.CreateOrUpdateRoleParams{
			IfUnmodifiedSince: &role.Metadata.ResourceVersion,
		}, *role, api.loadRequestHeaders)
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

func (api *AuthorizationV1) DeleteRole(ctx context.Context, role *authorization.Role) error {
	if err := validateAuthorizationMetadataV1(role.Metadata); err != nil {
		return err
	}

	resp, err := api.authorization.DeleteRoleWithResponse(ctx, role.Metadata.Tenant, role.Metadata.Name, &authorization.DeleteRoleParams{
		IfUnmodifiedSince: &role.Metadata.ResourceVersion,
	}, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
	}

	return nil
}

// Role Assignment

func (api *AuthorizationV1) ListRoleAssignments(ctx context.Context, tid TenantID) (*Iterator[authorization.RoleAssignment], error) {
	iter := Iterator[authorization.RoleAssignment]{
		fn: func(ctx context.Context, skipToken *string) ([]authorization.RoleAssignment, *string, error) {
			resp, err := api.authorization.ListRoleAssignmentsWithResponse(ctx, authorization.Tenant(tid), &authorization.ListRoleAssignmentsParams{
				Accept: ptr.To(authorization.ListRoleAssignmentsParamsAcceptApplicationjson),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *AuthorizationV1) GetRoleAssignment(ctx context.Context, tref TenantReference) (*authorization.RoleAssignment, error) {
	if err := validateTenantReference(tref); err != nil {
		return nil, err
	}

	resp, err := api.authorization.GetRoleAssignmentWithResponse(ctx, authorization.Tenant(tref.Tenant), string(tref.Name), api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

func (api *AuthorizationV1) CreateOrUpdateRoleAssignment(ctx context.Context, assign *authorization.RoleAssignment) (*authorization.RoleAssignment, error) {
	if err := validateAuthorizationMetadataV1(assign.Metadata); err != nil {
		return nil, err
	}

	resp, err := api.authorization.CreateOrUpdateRoleAssignmentWithResponse(ctx, assign.Metadata.Tenant, assign.Metadata.Name,
		&authorization.CreateOrUpdateRoleAssignmentParams{
			IfUnmodifiedSince: &assign.Metadata.ResourceVersion,
		}, *assign, api.loadRequestHeaders)
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

func (api *AuthorizationV1) DeleteRoleAssignment(ctx context.Context, assign *authorization.RoleAssignment) error {
	if err := validateAuthorizationMetadataV1(assign.Metadata); err != nil {
		return err
	}

	resp, err := api.authorization.DeleteRoleAssignmentWithResponse(ctx, assign.Metadata.Tenant, assign.Metadata.Name, &authorization.DeleteRoleAssignmentParams{
		IfUnmodifiedSince: &assign.Metadata.ResourceVersion,
	}, api.loadRequestHeaders)
	if err != nil {
		return err
	}

	if err = checkSuccessDeleteStatusCodes(resp); err != nil {
		return err
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

func validateAuthorizationMetadataV1(metadata *authorization.GlobalResourceMetadata) error {
	if metadata == nil {
		return ErrNoMetatada
	}

	if metadata.Tenant == "" {
		return ErrNoMetatadaTenant
	}

	return nil
}
