package crossplane

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

func ionosScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := ionosv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newPublicIP() *publicipdom.PublicIp {
	p := &publicipdom.PublicIp{}
	p.Name = "public-ip-1"
	p.Scope = resource.Scope{Tenant: "tenant-1", Workspace: "workspace-1"}
	return p
}

func TestPublicIPStoreCreateReservesIPBlock(t *testing.T) {
	c := fakeclient.NewClientBuilder().WithScheme(ionosScheme(t)).Build()
	store := NewPublicIPStore(c, testLogger())

	// First reconcile: creates the IPBlock and waits.
	if err := store.Create(context.Background(), newPublicIP()); !errors.Is(err, backend.ErrStillProcessing) {
		t.Fatalf("Create #1 = %v, want ErrStillProcessing", err)
	}

	ipb := &ionosv1alpha1.Ipblock{}
	// ComputeNamespace(&resource.Scope{Tenant: "tenant-1"}) hashes the tenant with
	// SHA3-224 rather than returning the literal tenant name; this is that hash.
	key := client.ObjectKey{Namespace: "18378ae6c422c5a5d47255bb89e7a44e3324c7fbcac9995a3ea588ec", Name: "public-ip-1"}
	if err := c.Get(context.Background(), key, ipb); err != nil {
		t.Fatalf("ipblock not created: %v", err)
	}
	if ipb.Spec.ForProvider.Size == nil || *ipb.Spec.ForProvider.Size != 1 {
		t.Fatalf("ipblock size = %v, want 1", ipb.Spec.ForProvider.Size)
	}
	if ipb.Spec.ForProvider.Location == nil || *ipb.Spec.ForProvider.Location != DefaultIPBlockLocation {
		t.Fatalf("ipblock location = %v, want %q", ipb.Spec.ForProvider.Location, DefaultIPBlockLocation)
	}
}
