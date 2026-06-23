package workspace

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

type DeleteWorkspace struct {
	Store port.WorkspaceStore
}

func (d *DeleteWorkspace) Do(ctx context.Context, domain *wsdom.Workspace) error {
	return d.Store.Delete(ctx, domain)
}
