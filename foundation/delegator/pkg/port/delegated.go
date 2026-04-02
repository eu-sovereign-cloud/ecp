package port

import (
	"context"
	"errors"

	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

var ErrStillProcessing = errors.New("operation still in progress")

type DelegatedFunc[T gateway.IdentifiableResource] func(ctx context.Context, resource T) error
