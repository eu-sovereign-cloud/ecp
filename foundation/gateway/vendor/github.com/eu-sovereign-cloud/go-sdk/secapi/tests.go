package secapi

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/eu-sovereign-cloud/go-sdk/internal/secatest"

	"github.com/stretchr/testify/require"
)

func newTestGlobalClientV1(t *testing.T, server *httptest.Server) *GlobalClient {
	config := &GlobalConfig{
		AuthToken: secatest.AuthToken,
		Endpoints: GlobalEndpoints{
			RegionV1:        server.URL + secatest.ProviderRegionEndpoint,
			AuthorizationV1: server.URL + secatest.ProviderAuthorizationEndpoint,
		},
	}
	client, err := NewGlobalClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)

	return client
}

func newTestRegionalClientV1(t *testing.T, ctx context.Context, server *httptest.Server) *RegionalClient {
	globalClient := newTestGlobalClientV1(t, server)

	regionalClient, err := globalClient.NewRegionalClient(ctx, secatest.Region1Name)
	require.NoError(t, err)
	require.NotNil(t, regionalClient)

	return regionalClient
}
