package port

import (
	"context"

	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

type NicStore interface {
	Create(ctx context.Context, domain *nicdom.Nic) error
	Delete(ctx context.Context, domain *nicdom.Nic) error
}
