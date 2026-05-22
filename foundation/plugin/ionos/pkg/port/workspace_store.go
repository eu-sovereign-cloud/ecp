package port

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

type WorkspaceStore interface {
	Create(ctx context.Context, domain *regional.WorkspaceDomain) error
	Delete(ctx context.Context, domain *regional.WorkspaceDomain) error
}
