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

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// fakeWorkspaceRepo is a hand-rolled ReaderRepo+WriterRepo for exercising the workspace
// Handler over HTTP. Create/Update capture the written domain object.
type fakeWorkspaceRepo struct {
	written    *wsdom.Workspace
	loaded     *wsdom.Workspace
	deleted    *wsdom.Workspace
	listParams resource.ListFilter
}

func (f *fakeWorkspaceRepo) List(_ context.Context, params resource.ListFilter, _ *[]*wsdom.Workspace) (*string, error) {
	f.listParams = params
	return nil, nil
}

func (f *fakeWorkspaceRepo) Load(_ context.Context, m **wsdom.Workspace) error {
	f.loaded = *m
	return nil
}

func (f *fakeWorkspaceRepo) Delete(_ context.Context, m *wsdom.Workspace) error {
	f.deleted = m
	return nil
}

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

// TestHandler_RegionIsPartOfTheIdentity covers the read and delete sides of issue #396: the
// region the request was addressed to travels into the lookup key, not just onto a created
// resource, so a GET, LIST or DELETE in one region can never resolve another region's
// workspace of the same name.
func TestHandler_RegionIsPartOfTheIdentity(t *testing.T) {
	newHandler := func(repo *fakeWorkspaceRepo) *Handler {
		return &Handler{Reader: repo, Writer: repo, Logger: slog.Default(), Region: "eu-central"}
	}
	request := func(method, region string) *http.Request {
		ctx := context.Background()
		if region != "" {
			ctx = resource.ContextWithRegion(ctx, region)
		}
		return httptest.NewRequestWithContext(ctx, method, "/", strings.NewReader("{}"))
	}

	t.Run("get", func(t *testing.T) {
		repo := &fakeWorkspaceRepo{}
		newHandler(repo).GetWorkspace(httptest.NewRecorder(), request(http.MethodGet, "us-west"), "t1", "ws1")

		require.NotNil(t, repo.loaded)
		require.Equal(t, "us-west", repo.loaded.Region)
	})

	t.Run("delete", func(t *testing.T) {
		repo := &fakeWorkspaceRepo{}
		newHandler(repo).DeleteWorkspace(httptest.NewRecorder(), request(http.MethodDelete, "us-west"), "t1", "ws1",
			sdkworkspace.DeleteWorkspaceParams{})

		require.NotNil(t, repo.deleted)
		require.Equal(t, "us-west", repo.deleted.Region)
	})

	// The list params carry the region so ReaderAdapter resolves the per-region namespace;
	// persistence.RegionScope is the interface it asserts for.
	t.Run("list", func(t *testing.T) {
		repo := &fakeWorkspaceRepo{}
		newHandler(repo).ListWorkspaces(httptest.NewRecorder(), request(http.MethodGet, "us-west"), "t1",
			sdkworkspace.ListWorkspacesParams{})

		regionScope, ok := repo.listParams.(persistence.RegionScope)
		require.True(t, ok, "workspace list params must implement persistence.RegionScope")
		require.Equal(t, "us-west", regionScope.GetRegion())
	})

	// A single-region gateway names no region per request and must keep resolving to its own.
	t.Run("falls back to the handler region", func(t *testing.T) {
		repo := &fakeWorkspaceRepo{}
		newHandler(repo).GetWorkspace(httptest.NewRecorder(), request(http.MethodGet, ""), "t1", "ws1")

		require.NotNil(t, repo.loaded)
		require.Equal(t, "eu-central", repo.loaded.Region)
	})
}
