//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

// TestPublicIPVersionSurvivesTheStack pins a spec enum against the four converters it crosses —
// API→domain at the gateway, domain→CR on the write, CR→domain in the delegator, domain→API on the
// read — which each used to answer the empty string for a value they did not recognise. Rejection
// is covered by the gateway integration suite; this is the only place all four run in one process
// tree.
//
// Like the update tests, this runs inside the workspace TestEndToEnd creates (file order puts it
// after flow_test.go and before update_test.go); TestMain tears that workspace down once, after
// the whole suite.
func TestPublicIPVersionSurvivesTheStack(t *testing.T) {
	ctx := context.Background()
	publicIPName := "e2e-pip-" + uuid.New().String()[:8]

	t.Cleanup(func() {
		testenv.DeleteUntilGone(ctx, func() (*http.Response, error) {
			return networkClient.DeletePublicIp(ctx, testTenant, testWorkspace, publicIPName, nil)
		})
	})

	body := schema.PublicIp{Spec: schema.PublicIpSpec{Version: schema.IPVersionIPv6}}
	resp, err := networkClient.CreateOrUpdatePublicIpWithResponse(ctx, testTenant, testWorkspace, publicIPName, nil, body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)
	require.Equal(t, schema.IPVersionIPv6, resp.JSON200.Spec.Version, "the create response must echo the version")

	waitForActive(t, "public ip", func(ctx context.Context) (schema.ResourceState, bool, error) {
		r, err := networkClient.GetPublicIpWithResponse(ctx, testTenant, testWorkspace, publicIPName)
		if err != nil {
			return "", false, err
		}
		if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
			return "", false, nil
		}
		return r.JSON200.Status.State, true, nil
	})

	// The read after reconciliation is the one that goes through CR→domain in the delegator's
	// own conversion and back out through the gateway.
	getResp, err := networkClient.GetPublicIpWithResponse(ctx, testTenant, testWorkspace, publicIPName)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, getResp.StatusCode())
	require.NotNil(t, getResp.JSON200)
	require.Equal(t, schema.IPVersionIPv6, getResp.JSON200.Spec.Version,
		"the version must survive the full API → CR → delegator → API round trip")
}
