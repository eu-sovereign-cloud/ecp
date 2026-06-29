package kubernetes

import (
	"context"

	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

// NicPlugin is implemented by CSP plugins that manage NIC resources.
type NicPlugin interface {
	Create(ctx context.Context, resource *nicdom.Nic) error
	Delete(ctx context.Context, resource *nicdom.Nic) error
}
