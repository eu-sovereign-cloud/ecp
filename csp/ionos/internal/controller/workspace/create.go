package workspace

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/pkg/port"
)

type CreateWorkspace struct {
	Store port.WorkspaceStore
}

func (c *CreateWorkspace) Do(ctx context.Context, domain *regional.WorkspaceDomain) error {
	return c.Store.Create(ctx, domain)
}
