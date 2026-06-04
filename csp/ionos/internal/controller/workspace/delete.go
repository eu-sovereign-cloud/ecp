package workspace

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
)

type DeleteWorkspace struct {
	Store port.WorkspaceStore
}

func (d *DeleteWorkspace) Do(ctx context.Context, domain *regional.WorkspaceDomain) error {
	return d.Store.Delete(ctx, domain)
}
