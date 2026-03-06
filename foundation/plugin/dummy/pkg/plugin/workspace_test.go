package plugin

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

func newTestWorkspace() *Workspace {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return newWorkspaceWithDelay(logger, noDelay)
}

func newWorkspaceDomain(name string) *regional.WorkspaceDomain {
	return &regional.WorkspaceDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{Name: name},
			Scope:          scope.Scope{Tenant: "test-tenant", Workspace: name},
		},
		Spec: regional.WorkspaceSpec{
			"description": "test workspace",
		},
	}
}

func TestWorkspace_Create(t *testing.T) {
	ws := newTestWorkspace()
	resource := newWorkspaceDomain("my-workspace")

	err := ws.Create(context.Background(), resource)
	require.NoError(t, err)
}

func TestWorkspace_Delete(t *testing.T) {
	ws := newTestWorkspace()
	resource := newWorkspaceDomain("my-workspace")

	err := ws.Delete(context.Background(), resource)
	require.NoError(t, err)
}

func TestNewWorkspace_UsesDefaultDelay(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ws := NewWorkspace(logger)
	assert.NotNil(t, ws)
	assert.NotNil(t, ws.delayFunc)
}
