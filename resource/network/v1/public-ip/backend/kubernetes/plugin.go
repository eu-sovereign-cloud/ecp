package kubernetes

import (
	"context"

	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

// PublicIpPlugin is implemented by CSP plugins that manage PublicIp resources.
type PublicIpPlugin interface {
	Create(ctx context.Context, resource *publicipdom.PublicIp) error
	Delete(ctx context.Context, resource *publicipdom.PublicIp) error
}
