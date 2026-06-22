package workspace

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/pkg/port"
)

type DeleteWorkspace struct {
	Store port.WorkspaceStore
}

func (d *DeleteWorkspace) Do(ctx context.Context, domain *regional.WorkspaceDomain) error {
	return d.Store.Delete(ctx, domain)
}
