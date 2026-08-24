//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// newSecurityGroupBody builds a security group carrying one inline rule with the given ip version.
func newSecurityGroupBody(version schema.IPVersion) schema.SecurityGroup {
	return schema.SecurityGroup{
		Spec: schema.SecurityGroupSpec{
			Rules: []schema.SecurityGroupRuleSpec{
				{
					Direction: schema.SecurityGroupRuleDirectionIngress,
					Protocol:  schema.SecurityGroupRuleProtocolTCP,
					Version:   version,
				},
			},
		},
	}
}

// TestSecurityGroupIPVersion covers the gateway's REST↔CR translation of an enum the CRD does not
// police: an inline rule's `version` is neither required nor enum-constrained in the security-group
// CRD, so it is the one field where a value the gateway does not recognise reaches storage.
//
// The gateway used to answer such a value with the empty string, and because nothing downstream
// rejected it the write succeeded — the caller got 200 for a rule the control plane had quietly
// stripped, and the following GET disagreed with what they sent.
func TestSecurityGroupIPVersion(t *testing.T) {
	t.Run("should preserve a valid ip version through create and read-back", func(t *testing.T) {
		//
		// Given a security group whose single rule is explicitly IPv6
		name := "test-sg-ipv6-" + uuid.New().String()[:8]
		t.Cleanup(func() {
			_, _ = networkClient.DeleteSecurityGroupWithResponse(context.Background(), testTenant, testWorkspace, name, nil)
		})

		//
		// When we create it through the gateway
		createResp, err := networkClient.CreateOrUpdateSecurityGroupWithResponse(
			context.Background(), testTenant, testWorkspace, name, nil, newSecurityGroupBody(schema.IPVersionIPv6),
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())
		require.NotNil(t, createResp.JSON200)
		require.Len(t, createResp.JSON200.Spec.Rules, 1)
		require.Equal(t, schema.IPVersionIPv6, createResp.JSON200.Spec.Rules[0].Version)

		//
		// Then a read-back returns the same version, not the empty string
		getResp, err := networkClient.GetSecurityGroupWithResponse(context.Background(), testTenant, testWorkspace, name)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode())
		require.NotNil(t, getResp.JSON200)
		require.Len(t, getResp.JSON200.Spec.Rules, 1)
		require.Equal(t, schema.IPVersionIPv6, getResp.JSON200.Spec.Rules[0].Version)
	})

	t.Run("should reject an unknown ip version instead of storing it as empty", func(t *testing.T) {
		//
		// Given a security group whose rule names an ip version that does not exist
		name := "test-sg-badver-" + uuid.New().String()[:8]
		t.Cleanup(func() {
			_, _ = networkClient.DeleteSecurityGroupWithResponse(context.Background(), testTenant, testWorkspace, name, nil)
		})

		//
		// When we create it through the gateway
		createResp, err := networkClient.CreateOrUpdateSecurityGroupWithResponse(
			context.Background(), testTenant, testWorkspace, name, nil, newSecurityGroupBody(schema.IPVersion("IPv9")),
		)
		require.NoError(t, err)

		//
		// Then the request is refused as a validation failure that names the value we sent,
		// rather than accepted with the value silently dropped.
		require.Equal(t, http.StatusUnprocessableEntity, createResp.StatusCode())
		require.NotNil(t, createResp.JSON422)
		require.Contains(t, createResp.JSON422.Detail, "IPv9",
			"the response must name the value the request carried, not a downstream field")

		//
		// And nothing was written.
		getResp, err := networkClient.GetSecurityGroupWithResponse(context.Background(), testTenant, testWorkspace, name)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, getResp.StatusCode())
	})
}
