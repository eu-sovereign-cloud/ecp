package crossplane

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	p.Region = "regionBerlin"
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
	if ipb.Spec.ForProvider.Location == nil || *ipb.Spec.ForProvider.Location != "de/txl" {
		t.Fatalf("ipblock location = %v, want %q", ipb.Spec.ForProvider.Location, "de/txl")
	}
}

// TestPublicIPStoreCreateRejectsUnknownRegion verifies that an unmapped SECA region fails fast
// instead of silently falling back to some default IONOS location, which would risk reserving
// the address somewhere the instance's workspace datacenter isn't.
func TestPublicIPStoreCreateRejectsUnknownRegion(t *testing.T) {
	c := fakeclient.NewClientBuilder().WithScheme(ionosScheme(t)).Build()
	store := NewPublicIPStore(c, testLogger())

	p := newPublicIP()
	p.Region = "regionAtlantis"
	if err := store.Create(context.Background(), p); err == nil || errors.Is(err, backend.ErrStillProcessing) {
		t.Fatalf("Create = %v, want a translation error for an unmapped region", err)
	}
}

// TestReadReservedIPSurfacesReconcileError verifies that a failed IPBlock (region out of
// addresses, quota exceeded, etc.) surfaces as an error instead of ErrStillProcessing forever —
// otherwise PowerOn would requeue indefinitely on an IPBlock the provider has already given up on.
func TestReadReservedIPSurfacesReconcileError(t *testing.T) {
	ns := "ns-1"
	ipb := &ionosv1alpha1.Ipblock{
		ObjectMeta: metav1.ObjectMeta{Name: "public-ip-1", Namespace: ns},
	}
	ipb.SetConditions(v1.ReconcileError(errors.New("no available addresses in de/txl")))
	c := fakeclient.NewClientBuilder().WithScheme(ionosScheme(t)).WithObjects(ipb).Build()

	_, err := readReservedIP(context.Background(), c, ns, "public-ip-1")
	if err == nil || errors.Is(err, backend.ErrStillProcessing) {
		t.Fatalf("readReservedIP = %v, want a non-ErrStillProcessing reconcile error", err)
	}
}
