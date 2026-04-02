package plugin

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

type Workspace interface {
	Create(ctx context.Context, resource *regional.WorkspaceDomain) error
	Delete(ctx context.Context, resource *regional.WorkspaceDomain) error
}
