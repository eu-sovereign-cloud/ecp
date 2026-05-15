package service

import (
	"context"

	delegatorplugin "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	workspacectrl "github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/internal/controller/workspace"
)

var _ delegatorplugin.Workspace = (*Workspace)(nil)

type Workspace struct {
	Creator *workspacectrl.CreateWorkspace
	Deleter *workspacectrl.DeleteWorkspace
}

func (s *Workspace) Create(ctx context.Context, resource *regional.WorkspaceDomain) error {
	return s.Creator.Do(ctx, resource)
}

func (s *Workspace) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	return s.Deleter.Do(ctx, resource)
}
