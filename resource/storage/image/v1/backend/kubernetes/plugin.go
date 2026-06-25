package kubernetes

import (
	"context"

	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1"
)

// ImagePlugin is implemented by CSP plugins that manage image resources.
type ImagePlugin interface {
	Create(ctx context.Context, resource *imgdom.Image) error
	Delete(ctx context.Context, resource *imgdom.Image) error
}
