package port

import (
	"context"

	gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type SetStateFunc[T gateway_port.IdentifiableResource] func(ctx context.Context, resource T) error
