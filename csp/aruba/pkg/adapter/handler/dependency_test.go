package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	refdom "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// TestLoadActiveWorkspaceAddressesTheRegionNamespace guards the one thing a stubbed repo
// otherwise hides: not that Load was called, but that the key handed to it resolves to the
// namespace the Workspace CR is actually stored in.
//
// A workspace is keyed by tenant AND region, so its CR lives in
// sha3-224("@region/<region>/<tenant>"). A key built without the region falls through to
// ComputeNamespace and addresses the plain tenant namespace instead, which holds no workspace:
// the read 404s, the handler reports StillProcessing, and every dependent Aruba resource
// requeues forever carrying no message about why. Nothing else catches that — every other test
// here stubs the repo, so the namespace is never resolved, and the e2e stack runs the dummy
// plugin, which has no dependency lookup at all.
func TestLoadActiveWorkspaceAddressesTheRegionNamespace(t *testing.T) {
	const (
		tenant = "tenant-1"
		region = "itbg-bergamo"
	)

	ctrl := gomock.NewController(t)
	repo := NewMockReaderRepo[*wsdom.Workspace](ctrl)

	// Capture the key rather than assert on a field: what matters is where it points.
	var key *wsdom.Workspace
	repo.EXPECT().
		Load(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ws **wsdom.Workspace) error {
			key = *ws
			(*ws).Status = &wsdom.WorkspaceStatus{
				Status: refdom.Status{State: refdom.ResourceStateActive},
			}
			return nil
		})

	scope := res.Scope{Tenant: tenant, Workspace: "ws-1"}
	_, err := loadActiveWorkspace(t.Context(), repo, &scope, region)
	require.NoError(t, err)
	require.NotNil(t, key, "the handler must have issued the read")

	namespace, err := k8sadapter.ResolveNamespace(key)
	require.NoError(t, err)

	expected := k8sadapter.ComputeRegionNamespace(&wsdom.Workspace{
		RegionalMetadata: refdom.RegionalMetadata{
			Region: region,
			Scope:  res.Scope{Tenant: tenant},
		},
	})
	require.Equal(t, expected, namespace,
		"the lookup must address the workspace's per-region namespace")
	require.NotEqual(t, k8sadapter.ComputeNamespace(&res.Scope{Tenant: tenant}), namespace,
		"a key without the region resolves the plain tenant namespace, where no workspace lives")
}
