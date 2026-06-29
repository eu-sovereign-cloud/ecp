package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
)

func TestRoleIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := RoleIteratorToAPI(nil, nil)
	require.Equal(t, "roles", iter.Metadata.Resource)
	require.Equal(t, "seca.authorization/v1", iter.Metadata.Provider)
}

func TestRoleToAPI_ResourceAndRef(t *testing.T) {
	r := roledom.Role{}
	r.Name = "role1"
	r.Tenant = "t1"
	r.Provider = roledom.ProviderID

	out := roleToAPI(r, "get")

	require.Equal(t, "roles/role1", out.Metadata.Resource)
	require.Equal(t, "seca.authorization/v1/tenants/t1/roles/role1", out.Metadata.Ref)
}
