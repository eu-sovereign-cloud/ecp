package workspace

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1"
)

type CreateWorkspace struct {
	Store port.WorkspaceStore
}

func (c *CreateWorkspace) Do(ctx context.Context, domain *wsdom.Workspace) error {
	return c.Store.Create(ctx, domain)
}
