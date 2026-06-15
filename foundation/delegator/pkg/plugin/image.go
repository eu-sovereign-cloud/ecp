package plugin

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
)

type Image interface {
	Create(ctx context.Context, resource *regional.ImageDomain) error
	Delete(ctx context.Context, resource *regional.ImageDomain) error
}
