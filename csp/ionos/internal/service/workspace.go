package service

import (
	"context"

	workspacectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/workspace"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

var _ wsk8s.WorkspacePlugin = (*Workspace)(nil)

type Workspace struct {
	Creator *workspacectrl.CreateWorkspace
	Deleter *workspacectrl.DeleteWorkspace
}

func (s *Workspace) Create(ctx context.Context, resource *wsdom.Workspace) error {
	return s.Creator.Do(ctx, resource)
}

func (s *Workspace) Delete(ctx context.Context, resource *wsdom.Workspace) error {
	return s.Deleter.Do(ctx, resource)
}
