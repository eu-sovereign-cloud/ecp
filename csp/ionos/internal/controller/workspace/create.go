package workspace

import (
	"context"

	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1/domain"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
)

type CreateWorkspace struct {
	Store port.WorkspaceStore
}

func (c *CreateWorkspace) Do(ctx context.Context, domain *wsdom.WorkspaceDomain) error {
	return c.Store.Create(ctx, domain)
}
