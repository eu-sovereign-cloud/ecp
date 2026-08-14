package port

import (
	"context"

	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

// SubnetStore reconciles the SECA subnet observer. In the POC no IONOS resource maps to a
// subnet (the public LAN covers it), so this is a pure declaration.
type SubnetStore interface {
	Create(ctx context.Context, domain *subnetdom.Subnet) error
	Delete(ctx context.Context, domain *subnetdom.Subnet) error
}
