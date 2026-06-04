package controller

import (
	"log/slog"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/workspace/v1"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin/handler"
	kubernetes2domain "github.com/eu-sovereign-cloud/ecp/foundation/models/converters/kubernetes2domain"
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
	maxConditions int,
) WorkspaceController {
	h := handler.NewWorkspacePluginHandler(repo, plugin)
	h.MaxConditions = maxConditions

	return (WorkspaceController)(NewGenericController[*regional.WorkspaceDomain](
		client,
		kubernetes2domain.MapCRToWorkspaceDomain,
		h,
		&workspacev1.Workspace{},
		requeueAfter,
		logger,
		maxConditions,
	))
}
