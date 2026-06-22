package workspace

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1"
)

type DeleteWorkspace struct {
	Store port.WorkspaceStore
}

func (d *DeleteWorkspace) Do(ctx context.Context, domain *wsdom.WorkspaceDomain) error {
	return d.Store.Delete(ctx, domain)
}
