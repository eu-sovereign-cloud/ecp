package port

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
)

type NetworkStore interface {
	Create(ctx context.Context, domain *regional.NetworkDomain) error
	Delete(ctx context.Context, domain *regional.NetworkDomain) error
}
