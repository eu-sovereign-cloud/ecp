package crossplane

import (
	"context"
	"log/slog"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

var _ port.NicStore = (*NicStore)(nil)

type NicStore struct {
	base
}

func NewNicStore(c client.Client, logger *slog.Logger) *NicStore {
	return &NicStore{base{client: c, logger: logger}}
}

// Create defers NIC provisioning: IONOS requires server attachment, which is
// only known once an Instance claims this NIC via its primaryNicRef.
func (a *NicStore) Create(_ context.Context, domain *nicdom.Nic) error {
	a.logger.Info("NIC provisioning deferred: awaiting instance attachment", "name", domain.GetName())
	return backend.ErrStillProcessing
}

// Delete is a no-op because Create never provisions an IONOS NIC CR.
func (a *NicStore) Delete(_ context.Context, _ *nicdom.Nic) error {
	return nil
}
