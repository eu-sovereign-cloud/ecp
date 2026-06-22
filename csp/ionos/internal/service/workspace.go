package service

import (
	"context"

	workspacectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/workspace"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1/domain"
)

var _ wsk8s.WorkspacePlugin = (*Workspace)(nil)

type Workspace struct {
	Creator *workspacectrl.CreateWorkspace
	Deleter *workspacectrl.DeleteWorkspace
}

func (s *Workspace) Create(ctx context.Context, resource *wsdom.WorkspaceDomain) error {
	return s.Creator.Do(ctx, resource)
}

func (s *Workspace) Delete(ctx context.Context, resource *wsdom.WorkspaceDomain) error {
	return s.Deleter.Do(ctx, resource)
}
