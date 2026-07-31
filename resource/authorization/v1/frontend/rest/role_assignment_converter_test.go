package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
)

func TestRoleAssignmentIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := roleAssignmentIteratorToAPI(nil, nil)
	// TODO_TEST_238_239
	// require.Equal(t, "role-assignments", iter.Metadata.Resource)
	require.Equal(t, "role-assignments", iter.Metadata.Resource)
	require.Equal(t, "seca.authorization/v1", iter.Metadata.Provider)
}

func TestRoleAssignmentToAPI_ResourceAndRef(t *testing.T) {
	ra := &radom.RoleAssignment{}
	ra.Name = "ra1"
	ra.Tenant = "t1"
	ra.Provider = radom.ProviderID

	out := roleAssignmentToAPI(ra)

	// TODO_TEST_238_239
	// require.Equal(t, "role-assignments/ra1", out.Metadata.Resource)
	require.Equal(t, "role-assignments/ra1", out.Metadata.Resource)
	// TODO_TEST_238_239
	// require.Equal(t, "seca.authorization/v1/tenants/t1/role-assignments/ra1", out.Metadata.Ref)
	require.Equal(t, "seca.authorization/v1/tenants/t1/role-assignments/ra1", out.Metadata.Ref)
}
