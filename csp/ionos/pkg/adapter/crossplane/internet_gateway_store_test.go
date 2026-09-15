package crossplane

import (
	"context"
	"testing"

	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// The internet gateway is a declaration only: IONOS has no gateway object, and the LAN the
// Network plugin creates is already Public=true. Create and Delete must therefore succeed
// immediately — never StillProcessing, which would requeue forever — and must not write a CR.
func TestInternetGatewayStoreIsDeclarationOnly(t *testing.T) {
	for _, egressOnly := range []bool{false, true} {
		c := fakeclient.NewClientBuilder().WithScheme(ionosScheme(t)).Build()
		store := NewInternetGatewayStore(c, testLogger())

		ig := &internetgatewaydom.InternetGateway{
			Spec: internetgatewaydom.InternetGatewaySpec{EgressOnly: egressOnly},
		}
		ig.Name = "internet-gateway-1"
		ig.Region = "regionBerlin"

		if err := store.Create(context.Background(), ig); err != nil {
			t.Fatalf("Create(egressOnly=%v) = %v, want nil", egressOnly, err)
		}
		if err := store.Delete(context.Background(), ig); err != nil {
			t.Fatalf("Delete(egressOnly=%v) = %v, want nil", egressOnly, err)
		}
	}
}
