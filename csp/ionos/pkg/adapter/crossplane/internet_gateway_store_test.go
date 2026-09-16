package crossplane

import (
	"context"
	"errors"
	"testing"

	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// The internet gateway is a declaration only: IONOS has no gateway object, and the LAN the
// Network plugin creates is already Public=true. Create and Delete must therefore succeed
// immediately — never StillProcessing, which would requeue forever — and must not write a CR.
func TestInternetGatewayStoreIsDeclarationOnly(t *testing.T) {
	c := fakeclient.NewClientBuilder().WithScheme(ionosScheme(t)).Build()
	store := NewInternetGatewayStore(c, testLogger())

	ig := &internetgatewaydom.InternetGateway{
		Spec: internetgatewaydom.InternetGatewaySpec{EgressOnly: false},
	}
	ig.Name = "internet-gateway-1"
	ig.Region = "regionBerlin"

	if err := store.Create(context.Background(), ig); err != nil {
		t.Fatalf("Create(egressOnly=false) = %v, want nil", err)
	}
	if err := store.Delete(context.Background(), ig); err != nil {
		t.Fatalf("Delete(egressOnly=false) = %v, want nil", err)
	}
}

// A public IONOS LAN is bidirectional; there is no per-LAN switch to block inbound traffic.
// Create must refuse an egressOnly request rather than report the gateway ready while still
// admitting inbound traffic to any NIC holding a public IP.
func TestInternetGatewayStoreRejectsEgressOnly(t *testing.T) {
	c := fakeclient.NewClientBuilder().WithScheme(ionosScheme(t)).Build()
	store := NewInternetGatewayStore(c, testLogger())

	ig := &internetgatewaydom.InternetGateway{
		Spec: internetgatewaydom.InternetGatewaySpec{EgressOnly: true},
	}
	ig.Name = "internet-gateway-1"
	ig.Region = "regionBerlin"

	err := store.Create(context.Background(), ig)
	if !errors.Is(err, backend.ErrNotSupported) {
		t.Fatalf("Create(egressOnly=true) = %v, want error wrapping backend.ErrNotSupported", err)
	}
}
