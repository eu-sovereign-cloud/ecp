package plugin

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
)

type Network interface {
	Create(ctx context.Context, resource *regional.NetworkDomain) error
	Delete(ctx context.Context, resource *regional.NetworkDomain) error
}
