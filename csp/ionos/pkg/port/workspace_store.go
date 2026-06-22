package port

import (
	"context"

	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1"
)

type WorkspaceStore interface {
	Create(ctx context.Context, domain *wsdom.WorkspaceDomain) error
	Delete(ctx context.Context, domain *wsdom.WorkspaceDomain) error
}
