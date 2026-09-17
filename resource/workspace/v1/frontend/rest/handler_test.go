package rest

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// fakeWorkspaceRepo is a hand-rolled ReaderRepo+WriterRepo for exercising the workspace
// Handler over HTTP. Create/Update capture the written domain object.
type fakeWorkspaceRepo struct {
	written *wsdom.Workspace
}

func (f *fakeWorkspaceRepo) List(context.Context, resource.ListFilter, *[]*wsdom.Workspace) (*string, error) {
	return nil, nil
}

func (f *fakeWorkspaceRepo) Load(context.Context, **wsdom.Workspace) error {
	return nil
}

func (f *fakeWorkspaceRepo) Delete(context.Context, *wsdom.Workspace) error { return nil }

func (f *fakeWorkspaceRepo) Create(_ context.Context, m *wsdom.Workspace) (**wsdom.Workspace, error) {
	f.written = m
	return &m, nil
}

func (f *fakeWorkspaceRepo) Update(_ context.Context, m *wsdom.Workspace) (**wsdom.Workspace, error) {
	f.written = m
	return &m, nil
}

func (f *fakeWorkspaceRepo) UpdateStatus(_ context.Context, m *wsdom.Workspace) (**wsdom.Workspace, error) {
	return &m, nil
}

// TestHandler_CreateOrUpdateWorkspace_UsesInjectedRegion verifies that the Handler's Region
// field (not a process-global singleton) is what determines the region stamped on a created
// workspace. Constructing two Handlers with different Region values in the same test process
// — something the former sync.Once singleton could not support — proves the dependency is
// now truly per-handler.
func TestHandler_CreateOrUpdateWorkspace_UsesInjectedRegion(t *testing.T) {
	for _, region := range []string{"eu-central", "us-west"} {
		t.Run(region, func(t *testing.T) {
			repo := &fakeWorkspaceRepo{}
			h := &Handler{
				Reader: repo,
				Writer: repo,
				Logger: slog.Default(),
				Region: region,
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/", strings.NewReader("{}"))
			h.CreateOrUpdateWorkspace(rec, req, "t1", "ws1", sdkworkspace.CreateOrUpdateWorkspaceParams{})

			require.NotNil(t, repo.written)
			require.Equal(t, region, repo.written.Region)
		})
	}
}

// TestHandler_CreateOrUpdateWorkspace_RequestRegionWins verifies that when the gateway
// serves several regions, the region resolved for the request — not the handler's default —
// is what gets stamped on the created workspace, so the same Handler can place resources in
// whichever region the caller addressed.
func TestHandler_CreateOrUpdateWorkspace_RequestRegionWins(t *testing.T) {
	repo := &fakeWorkspaceRepo{}
	h := &Handler{Reader: repo, Writer: repo, Logger: slog.Default(), Region: "eu-central"}

	ctx := resource.ContextWithRegion(context.Background(), "us-west")
	req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/", strings.NewReader("{}"))
	h.CreateOrUpdateWorkspace(httptest.NewRecorder(), req, "t1", "ws1", sdkworkspace.CreateOrUpdateWorkspaceParams{})

	require.NotNil(t, repo.written)
	require.Equal(t, "us-west", repo.written.Region)
}
