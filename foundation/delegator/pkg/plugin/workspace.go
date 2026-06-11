package plugin

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
)

type Workspace interface {
	Create(ctx context.Context, resource *regional.WorkspaceDomain) error
	Delete(ctx context.Context, resource *regional.WorkspaceDomain) error
}
