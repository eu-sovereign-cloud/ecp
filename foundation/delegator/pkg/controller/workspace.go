package controller

import (
	"log/slog"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin/handler"
)

// WorkspaceController is the specialized controller for Workspace resources.
// It uses a GenericController as its base and is configured with the specific
// types and handlers for Workspace.
type WorkspaceController GenericController[*regional.WorkspaceDomain]

// NewWorkspaceController creates a new controller for Workspace resources.
func NewWorkspaceController(
	client client.Client,
	repo gateway.Repo[*regional.WorkspaceDomain],
	plugin plugin.Workspace,
	requeueAfter time.Duration,
	logger *slog.Logger,
) *WorkspaceController {
	return (*WorkspaceController)(NewGenericController[*regional.WorkspaceDomain](
		client,
		kubernetes.MapCRToWorkspaceDomain,
		handler.NewWorkspacePluginHandler(repo, plugin),
		&workspacev1.Workspace{},
		requeueAfter,
		logger,
	))
}
