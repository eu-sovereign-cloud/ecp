package crossplane

import (
	"context"
	"fmt"
	"log/slog"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

var _ port.InternetGatewayStore = (*InternetGatewayStore)(nil)

type InternetGatewayStore struct {
	base
}

func NewInternetGatewayStore(c client.Client, logger *slog.Logger) *InternetGatewayStore {
	return &InternetGatewayStore{base{client: c, logger: logger}}
}

// Create marks the internet gateway ready. IONOS has no gateway object: internet access is the
// Lan.ForProvider.Public bool, which the Network plugin always sets to true, so the network the
// gateway is attached to already routes to the internet. The gateway only needs to exist as a
// ready declaration.
//
// Spec.EgressOnly cannot be honoured. A public IONOS LAN is bidirectional, and there is no
// per-LAN switch to drop inbound traffic; ingress is instead gated by whether a NIC holds a
// public IP. Reporting the gateway ready anyway would tell the tenant their outgoing-only
// request took effect when inbound traffic to a publicly-addressed NIC still gets through, so
// this refuses instead of silently downgrading the guarantee.
func (a *InternetGatewayStore) Create(ctx context.Context, domain *internetgatewaydom.InternetGateway) error {
	if domain.Spec.EgressOnly {
		return fmt.Errorf("%w: egressOnly cannot be enforced on IONOS, a public LAN is bidirectional and inbound is gated only by public IP assignment", backend.ErrNotSupported)
	}
	a.logger.Info("internet gateway: ready as declaration, no IONOS resource (LANs are created with Public=true)",
		"internet_gateway", domain.GetName())
	return nil
}

// Delete is a no-op: there is no backing IONOS resource to remove. The LAN keeps Public=true; it
// is owned by the Network resource and torn down with it.
func (a *InternetGatewayStore) Delete(ctx context.Context, domain *internetgatewaydom.InternetGateway) error {
	return nil
}
