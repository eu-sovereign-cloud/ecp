package port

import (
	"context"

	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type DelegatedFunc[T gateway.IdentifiableResource] func(ctx context.Context, resource T) error
