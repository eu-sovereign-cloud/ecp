package port

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

type NetworkStore interface {
	Create(ctx context.Context, domain *regional.NetworkDomain) error
	Delete(ctx context.Context, domain *regional.NetworkDomain) error
}
