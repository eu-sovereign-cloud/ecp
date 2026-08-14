package port

import (
	"context"

	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

type PublicIPStore interface {
	Create(ctx context.Context, domain *publicipdom.PublicIp) error
	Delete(ctx context.Context, domain *publicipdom.PublicIp) error
}
