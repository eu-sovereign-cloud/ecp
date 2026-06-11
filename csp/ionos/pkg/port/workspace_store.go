package port

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
)

type WorkspaceStore interface {
	Create(ctx context.Context, domain *regional.WorkspaceDomain) error
	Delete(ctx context.Context, domain *regional.WorkspaceDomain) error
}
