package crossplane

import (
	"context"
	"log/slog"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

var _ port.SubnetStore = (*SubnetStore)(nil)

type SubnetStore struct {
	base
}

func NewSubnetStore(c client.Client, logger *slog.Logger) *SubnetStore {
	return &SubnetStore{base{client: c, logger: logger}}
}

// Create marks the subnet ready. For the POC the IONOS public LAN (created by the Network
// plugin) covers subnetting, so no IONOS resource maps to a SECA subnet. The Instance plugin
// resolves nic -> subnet -> network -> LAN by reading this CR's references, so the subnet only
// needs to exist as a ready declaration.
func (a *SubnetStore) Create(ctx context.Context, domain *subnetdom.Subnet) error {
	a.logger.Info("subnet: ready as declaration, no IONOS resource (folded into the public LAN)",
		"subnet", domain.GetName())
	return nil
}

// Delete is a no-op: there is no backing IONOS resource to remove.
func (a *SubnetStore) Delete(ctx context.Context, domain *subnetdom.Subnet) error {
	return nil
}
