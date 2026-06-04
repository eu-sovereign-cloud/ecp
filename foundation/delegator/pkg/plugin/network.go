package plugin

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

type Network interface {
	Create(ctx context.Context, resource *regional.NetworkDomain) error
	Delete(ctx context.Context, resource *regional.NetworkDomain) error
}
