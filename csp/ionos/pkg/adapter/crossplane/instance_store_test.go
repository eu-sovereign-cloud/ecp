package crossplane

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	skuk8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku/backend/kubernetes"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
)

func instanceScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := ionosv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := skuk8s.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

// fullInstanceScheme additionally registers the ECP CRDs (Image, BlockStorage, NIC) that
// PowerOn's readBootImageAlias/readNicNetworking read, for tests exercising PowerOn past
// the Server-creation step.
func fullInstanceScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := instanceScheme(t)
	if err := imgk8s.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := bsk8s.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := nick8s.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func readyDatacenter(ns string) *ionosv1alpha1.Datacenter {
	dc := &ionosv1alpha1.Datacenter{
		ObjectMeta: metav1.ObjectMeta{Name: "workspace-1", Namespace: ns, Generation: 1},
	}
	dc.SetConditions(v1.Available().WithObservedGeneration(1))
	return dc
}

func testInstance() *instancedom.Instance {
	i := &instancedom.Instance{}
	i.Name = "instance-1"
	i.Scope = resource.Scope{Tenant: "tenant-1", Workspace: "workspace-1"}
	i.Spec.Zone = "a"
	i.Spec.SkuRef = commondomain.Reference{Resource: "sku/DXS"}
	i.Spec.SshKeys = []string{"ssh-ed25519 AAAA... example@secapi.cloud"}
	i.Spec.BootVolume = instancedom.VolumeReference{DeviceRef: commondomain.Reference{Resource: "block-storage/block-storage-1"}}
	i.Spec.PrimaryNicRef = &commondomain.Reference{Resource: "nic/nic-1"}
	return i
}

// First PowerOn creates the Server SHUTOFF and waits — it must not boot before its boot
// volume and NIC are attached.
func TestPowerOnCreatesServerFirst(t *testing.T) {
	dcNs := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: "tenant-1"})
	ns := instanceNamespace(testInstance())

	sku := &skuk8s.InstanceSKU{
		ObjectMeta: metav1.ObjectMeta{Name: "DXS", Namespace: dcNs},
		Spec:       skuk8s.InstanceSkuSpec{VCPU: 2, Ram: 4},
	}
	c := fakeclient.NewClientBuilder().
		WithScheme(instanceScheme(t)).
		WithObjects(readyDatacenter(dcNs), sku).
		Build()
	store := NewInstanceStore(c, testLogger())

	if err := store.PowerOn(context.Background(), testInstance()); !errors.Is(err, backend.ErrStillProcessing) {
		t.Fatalf("PowerOn = %v, want ErrStillProcessing", err)
	}

	srv := &ionosv1alpha1.Server{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "instance-1"}, srv); err != nil {
		t.Fatalf("server not created: %v", err)
	}
	if srv.Spec.ForProvider.Name == nil || *srv.Spec.ForProvider.Name != "instance-1" {
		t.Fatalf("server name = %v, want instance-1", srv.Spec.ForProvider.Name)
	}
	if srv.Spec.ForProvider.Type == nil || *srv.Spec.ForProvider.Type != "ENTERPRISE" {
		t.Fatalf("server type = %v, want ENTERPRISE", srv.Spec.ForProvider.Type)
	}
	if srv.Spec.ForProvider.Cores == nil || *srv.Spec.ForProvider.Cores != 2 {
		t.Fatalf("server cores = %v, want 2", srv.Spec.ForProvider.Cores)
	}
	if srv.Spec.ForProvider.RAM == nil || *srv.Spec.ForProvider.RAM != 4096 {
		t.Fatalf("server ram = %v, want 4096", srv.Spec.ForProvider.RAM)
	}
	if srv.Spec.ForProvider.VMState == nil || *srv.Spec.ForProvider.VMState != serverVMStateShutoff {
		t.Fatalf("server vmState = %v, want SHUTOFF (must not boot before the boot volume is attached)", srv.Spec.ForProvider.VMState)
	}

	// Boot volume and NIC must not have been created yet: the Server isn't even Ready.
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "block-storage-1"}, &ionosv1alpha1.Volume{}); err == nil {
		t.Fatal("boot volume should not be created before the server is ready")
	}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "nic-1"}, &ionosv1alpha1.Nic{}); err == nil {
		t.Fatal("nic should not be created before the server is ready")
	}
}

// Once the Server exists and is Ready (but still SHUTOFF), PowerOn must attach the boot
// volume and NIC before flipping it to RUNNING — reproduces the bug where the Server was
// brought RUNNING before its boot device existed, so it booted with no boot volume even
// though PowerOn went on to report success once the volume/NIC eventually became Ready.
func TestPowerOnCreatesBootVolumeAndNicWhileServerStillShutoff(t *testing.T) {
	dcNs := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: "tenant-1"})
	ns := instanceNamespace(testInstance())

	sku := &skuk8s.InstanceSKU{
		ObjectMeta: metav1.ObjectMeta{Name: "DXS", Namespace: dcNs},
		Spec:       skuk8s.InstanceSkuSpec{VCPU: 2, Ram: 4},
	}
	srv := &ionosv1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{Name: "instance-1", Namespace: ns, Generation: 1},
		Spec: ionosv1alpha1.ServerSpec{
			ForProvider: ionosv1alpha1.ServerParameters{Name: new("instance-1"), VMState: new(serverVMStateShutoff)},
		},
	}
	srv.SetConditions(v1.Available().WithObservedGeneration(1))

	img := &imgdom.Image{}
	img.Name = "image-1"
	img.Scope = resource.Scope{Tenant: "tenant-1"}
	img.Labels = map[string]string{"base": "ubuntu", "version": "24.04"}
	imgCR, err := imgk8s.ImageToCR(img)
	if err != nil {
		t.Fatalf("ImageToCR: %v", err)
	}

	bs := &bsdom.BlockStorage{}
	bs.Name = "block-storage-1"
	bs.Scope = resource.Scope{Tenant: "tenant-1", Workspace: "workspace-1"}
	bs.Spec.SizeGB = 42
	bs.Spec.SourceImageRef = &commondomain.Reference{Resource: "image/image-1"}
	bsCR, err := bsk8s.BlockStorageToCR(bs)
	if err != nil {
		t.Fatalf("BlockStorageToCR: %v", err)
	}

	nic := &nicdom.Nic{}
	nic.Name = "nic-1"
	nic.Scope = resource.Scope{Tenant: "tenant-1", Workspace: "workspace-1"}
	nic.Spec.SubnetRef = commondomain.Reference{Resource: "networks/lan-1/subnets/subnet-1"}
	nicCR, err := nick8s.NicToCR(nic)
	if err != nil {
		t.Fatalf("NicToCR: %v", err)
	}

	c := fakeclient.NewClientBuilder().
		WithScheme(fullInstanceScheme(t)).
		WithObjects(readyDatacenter(dcNs), sku, srv, imgCR, bsCR, nicCR).
		Build()
	store := NewInstanceStore(c, testLogger())

	if err := store.PowerOn(context.Background(), testInstance()); !errors.Is(err, backend.ErrStillProcessing) {
		t.Fatalf("PowerOn = %v, want ErrStillProcessing (boot volume/nic just created)", err)
	}

	// The server must still be SHUTOFF: the boot volume/NIC just got created and aren't
	// Ready yet, so PowerOn must not have flipped it to RUNNING.
	gotSrv := &ionosv1alpha1.Server{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "instance-1"}, gotSrv); err != nil {
		t.Fatal(err)
	}
	if gotSrv.Spec.ForProvider.VMState == nil || *gotSrv.Spec.ForProvider.VMState != serverVMStateShutoff {
		t.Fatalf("server vmState = %v, want still SHUTOFF while boot volume/nic aren't ready", gotSrv.Spec.ForProvider.VMState)
	}

	// The boot volume was just created (createCR returns ErrStillProcessing on fresh
	// creation), so PowerOn stops there for this reconcile — it hasn't reached the NIC
	// step yet. That's fine: each step is independently idempotent and re-driven by the
	// controller's requeue on ErrStillProcessing.
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "block-storage-1"}, &ionosv1alpha1.Volume{}); err != nil {
		t.Fatalf("boot volume should have been created: %v", err)
	}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "nic-1"}, &ionosv1alpha1.Nic{}); err == nil {
		t.Fatal("nic should not be created yet: PowerOn stops at the not-yet-ready boot volume this reconcile")
	}
}

// Once the boot Volume and NIC are both attached and Ready, PowerOn must flip the Server to
// RUNNING as its final step — the other half of the ordering fix: a Server must never boot
// before its boot device is attached, but once it's attached and ready there's nothing left
// to gate on.
func TestPowerOnFlipsToRunningOnlyOnceBootVolumeAndNicReady(t *testing.T) {
	dcNs := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: "tenant-1"})
	ns := instanceNamespace(testInstance())

	sku := &skuk8s.InstanceSKU{
		ObjectMeta: metav1.ObjectMeta{Name: "DXS", Namespace: dcNs},
		Spec:       skuk8s.InstanceSkuSpec{VCPU: 2, Ram: 4},
	}
	srv := &ionosv1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{Name: "instance-1", Namespace: ns, Generation: 1},
		Spec: ionosv1alpha1.ServerSpec{
			ForProvider: ionosv1alpha1.ServerParameters{Name: new("instance-1"), VMState: new(serverVMStateShutoff)},
		},
	}
	srv.SetConditions(v1.Available().WithObservedGeneration(1))

	img := &imgdom.Image{}
	img.Name = "image-1"
	img.Scope = resource.Scope{Tenant: "tenant-1"}
	img.Labels = map[string]string{"base": "ubuntu", "version": "24.04"}
	imgCR, err := imgk8s.ImageToCR(img)
	if err != nil {
		t.Fatalf("ImageToCR: %v", err)
	}

	bs := &bsdom.BlockStorage{}
	bs.Name = "block-storage-1"
	bs.Scope = resource.Scope{Tenant: "tenant-1", Workspace: "workspace-1"}
	bs.Spec.SizeGB = 42
	bs.Spec.SourceImageRef = &commondomain.Reference{Resource: "image/image-1"}
	bsCR, err := bsk8s.BlockStorageToCR(bs)
	if err != nil {
		t.Fatalf("BlockStorageToCR: %v", err)
	}

	nic := &nicdom.Nic{}
	nic.Name = "nic-1"
	nic.Scope = resource.Scope{Tenant: "tenant-1", Workspace: "workspace-1"}
	nic.Spec.SubnetRef = commondomain.Reference{Resource: "networks/lan-1/subnets/subnet-1"}
	nicCR, err := nick8s.NicToCR(nic)
	if err != nil {
		t.Fatalf("NicToCR: %v", err)
	}

	// The IONOS boot Volume already exists and is Ready. createCR's AlreadyExists path
	// (checkExisting) only checks readiness, not spec, so its ForProvider content doesn't
	// need to match newBootVolume's output exactly.
	vol := &ionosv1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{Name: "block-storage-1", Namespace: ns, Generation: 1},
	}
	vol.SetConditions(v1.Available().WithObservedGeneration(1))

	// The IONOS NIC already exists, Ready, and matching newNic's desired spec exactly —
	// ensureNic diffs spec fields and only leaves it alone (checkExisting) when they match.
	ionosNic := &ionosv1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{Name: "nic-1", Namespace: ns, Generation: 1},
		Spec: ionosv1alpha1.NicSpec{
			ForProvider: ionosv1alpha1.NicParameters_2{
				Name:            new("nic-1"),
				DatacenterIDRef: &v1.NamespacedReference{Name: "workspace-1", Namespace: dcNs},
				ServerIDRef:     &v1.NamespacedReference{Name: "instance-1", Namespace: ns},
				LanRef:          &v1.NamespacedReference{Name: "lan-1", Namespace: dcNs},
				DHCP:            new(true),
				FirewallActive:  new(false),
			},
		},
	}
	ionosNic.SetConditions(v1.Available().WithObservedGeneration(1))

	c := fakeclient.NewClientBuilder().
		WithScheme(fullInstanceScheme(t)).
		WithObjects(readyDatacenter(dcNs), sku, srv, imgCR, bsCR, nicCR, vol, ionosNic).
		Build()
	store := NewInstanceStore(c, testLogger())

	if err := store.PowerOn(context.Background(), testInstance()); !errors.Is(err, backend.ErrStillProcessing) {
		t.Fatalf("PowerOn = %v, want ErrStillProcessing (server VMState just flipped to RUNNING)", err)
	}

	gotSrv := &ionosv1alpha1.Server{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "instance-1"}, gotSrv); err != nil {
		t.Fatal(err)
	}
	if gotSrv.Spec.ForProvider.VMState == nil || *gotSrv.Spec.ForProvider.VMState != serverVMStateRunning {
		t.Fatalf("server vmState = %v, want RUNNING now that the boot volume and nic are ready", gotSrv.Spec.ForProvider.VMState)
	}
}

// A stopped-then-restarted instance's Server CR already exists (SHUTOFF, Ready) from the
// prior PowerOff. PowerOn must flip it back to RUNNING, not treat "already exists and ready"
// as "nothing to do" — reproduces a real restart getting stuck: the domain layer accepted the
// `start` action and recorded powerState "on", but the underlying IONOS Server stayed SHUTOFF
// forever because createCR's AlreadyExists fallback (checkExisting) only checks the Ready
// condition and never compares/updates VMState.
func TestPowerOnRestartsStoppedServer(t *testing.T) {
	ns := instanceNamespace(testInstance())

	srv := &ionosv1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{Name: "instance-1", Namespace: ns, Generation: 1},
		Spec: ionosv1alpha1.ServerSpec{
			ForProvider: ionosv1alpha1.ServerParameters{Name: new("instance-1"), VMState: new(serverVMStateShutoff)},
		},
	}
	srv.SetConditions(v1.Available().WithObservedGeneration(1))
	c := fakeclient.NewClientBuilder().WithScheme(instanceScheme(t)).WithObjects(srv).Build()
	store := NewInstanceStore(c, testLogger())

	if err := store.ensureServerRunning(context.Background(), testInstance(), ns); !errors.Is(err, backend.ErrStillProcessing) {
		t.Fatalf("ensureServerRunning = %v, want ErrStillProcessing", err)
	}

	got := &ionosv1alpha1.Server{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "instance-1"}, got); err != nil {
		t.Fatal(err)
	}
	if got.Spec.ForProvider.VMState == nil || *got.Spec.ForProvider.VMState != serverVMStateRunning {
		t.Fatalf("vmState = %v, want RUNNING", got.Spec.ForProvider.VMState)
	}
}

// Two workspaces in the same tenant can both name an instance "web" (instance names are only
// workspace-unique). PowerOn for the second workspace's "web" must create its OWN Server, not
// find and take over the first workspace's — reproduces a real collision where the second
// PowerOn's ensureServer Get matched the first workspace's Server object (same tenant-only
// namespace, same name) and started reconciling/powering it on without ever checking whose
// datacenter it actually belongs to.
func TestPowerOnDoesNotCollideAcrossWorkspaces(t *testing.T) {
	instanceA := testInstance() // tenant-1/workspace-1/instance-1
	instanceB := testInstance()
	instanceB.Workspace = "workspace-2"

	nsA := instanceNamespace(instanceA)
	nsB := instanceNamespace(instanceB)
	if nsA == nsB {
		t.Fatalf("instanceNamespace must differ across workspaces, got the same %q for both", nsA)
	}

	dcNs := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: "tenant-1"})
	srvA := &ionosv1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{Name: "instance-1", Namespace: nsA, Generation: 1},
		Spec: ionosv1alpha1.ServerSpec{
			ForProvider: ionosv1alpha1.ServerParameters{Name: new("instance-1"), VMState: new(serverVMStateRunning)},
		},
	}
	srvA.SetConditions(v1.Available().WithObservedGeneration(1))

	sku := &skuk8s.InstanceSKU{
		ObjectMeta: metav1.ObjectMeta{Name: "DXS", Namespace: dcNs},
		Spec:       skuk8s.InstanceSkuSpec{VCPU: 2, Ram: 4},
	}
	dcB := &ionosv1alpha1.Datacenter{
		ObjectMeta: metav1.ObjectMeta{Name: "workspace-2", Namespace: dcNs, Generation: 1},
	}
	dcB.SetConditions(v1.Available().WithObservedGeneration(1))

	c := fakeclient.NewClientBuilder().
		WithScheme(instanceScheme(t)).
		WithObjects(srvA, dcB, sku).
		Build()
	store := NewInstanceStore(c, testLogger())

	if err := store.PowerOn(context.Background(), instanceB); !errors.Is(err, backend.ErrStillProcessing) {
		t.Fatalf("PowerOn = %v, want ErrStillProcessing (workspace-2's own Server just created)", err)
	}

	// workspace-1's Server must be untouched (still RUNNING, unchanged).
	gotA := &ionosv1alpha1.Server{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: nsA, Name: "instance-1"}, gotA); err != nil {
		t.Fatalf("workspace-1's server should still exist: %v", err)
	}
	if gotA.Spec.ForProvider.VMState == nil || *gotA.Spec.ForProvider.VMState != serverVMStateRunning {
		t.Fatalf("workspace-1's server vmState = %v, want unchanged RUNNING", gotA.Spec.ForProvider.VMState)
	}

	// workspace-2 must have its own, separate Server, created SHUTOFF (freshly provisioned).
	gotB := &ionosv1alpha1.Server{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: nsB, Name: "instance-1"}, gotB); err != nil {
		t.Fatalf("workspace-2's own server should have been created: %v", err)
	}
	if gotB.Spec.ForProvider.VMState == nil || *gotB.Spec.ForProvider.VMState != serverVMStateShutoff {
		t.Fatalf("workspace-2's server vmState = %v, want SHUTOFF (freshly created, distinct from workspace-1's)", gotB.Spec.ForProvider.VMState)
	}
	if gotB.Spec.ForProvider.DatacenterIDRef == nil || gotB.Spec.ForProvider.DatacenterIDRef.Name != "workspace-2" {
		t.Fatalf("workspace-2's server datacenterIDRef = %v, want workspace-2", gotB.Spec.ForProvider.DatacenterIDRef)
	}
}

func TestPowerOffSetsShutoff(t *testing.T) {
	ns := instanceNamespace(testInstance())

	srv := &ionosv1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{Name: "instance-1", Namespace: ns, Generation: 1},
		Spec: ionosv1alpha1.ServerSpec{
			ForProvider: ionosv1alpha1.ServerParameters{VMState: new(serverVMStateRunning)},
		},
	}
	c := fakeclient.NewClientBuilder().WithScheme(instanceScheme(t)).WithObjects(srv).Build()
	store := NewInstanceStore(c, testLogger())

	_ = store.PowerOff(context.Background(), testInstance()) // returns ErrStillProcessing while applying

	got := &ionosv1alpha1.Server{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "instance-1"}, got); err != nil {
		t.Fatal(err)
	}
	if got.Spec.ForProvider.VMState == nil || *got.Spec.ForProvider.VMState != serverVMStateShutoff {
		t.Fatalf("vmState = %v, want SHUTOFF", got.Spec.ForProvider.VMState)
	}
}

func TestDeleteRemovesServer(t *testing.T) {
	ns := instanceNamespace(testInstance())

	srv := &ionosv1alpha1.Server{ObjectMeta: metav1.ObjectMeta{Name: "instance-1", Namespace: ns}}
	c := fakeclient.NewClientBuilder().WithScheme(instanceScheme(t)).WithObjects(srv).Build()
	store := NewInstanceStore(c, testLogger())

	// Nic and Volume are already gone (not created); Delete should progress to the server.
	_ = store.Delete(context.Background(), testInstance())

	got := &ionosv1alpha1.Server{}
	err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "instance-1"}, got)
	if err == nil && got.GetDeletionTimestamp().IsZero() {
		t.Fatal("server was not deleted")
	}
}

// TestNewNicIpsOmittedWithoutPublicIP verifies newNic leaves Ips unset (nil) when no
// public IP was reserved (DHCP fallback), and sets it to a single-element slice when one
// was resolved by readNicNetworking.
func TestNewNicIpsOmittedWithoutPublicIP(t *testing.T) {
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: "tenant-1"})
	store := NewInstanceStore(nil, testLogger())
	domainInst := testInstance()

	nic := store.newNic(domainInst, ns, "nic-1", "lan-1", "")
	if nic.Spec.ForProvider.Ips != nil {
		t.Fatalf("newNic Ips = %v, want nil (DHCP fallback)", nic.Spec.ForProvider.Ips)
	}

	nicWithIP := store.newNic(domainInst, ns, "nic-1", "lan-1", "203.0.113.10")
	if len(nicWithIP.Spec.ForProvider.Ips) != 1 || nicWithIP.Spec.ForProvider.Ips[0] == nil || *nicWithIP.Spec.ForProvider.Ips[0] != "203.0.113.10" {
		t.Fatalf("newNic Ips = %v, want [203.0.113.10]", nicWithIP.Spec.ForProvider.Ips)
	}
}

// The upjet IONOS provider late-inits whatever address IONOS currently reports into
// spec.forProvider.ips, on its own reconcile schedule, entirely outside our control. When the
// domain has no reserved public IP (DHCP fallback), PowerOn must NOT try to fight that by forcing
// ips back to nil: doing so races the provider's own late-init and never converges, which blocks
// ensureNic (and therefore PowerOn) from ever returning nil — the instance's power state gets
// stuck "off" forever even though the real Server/Nic are up and reachable. DHCP fallback must
// accept whatever address is currently present, ready or not.
func TestPowerOnAcceptsDHCPAssignedNicIP(t *testing.T) {
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: "tenant-1"})

	nic := &ionosv1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{Name: "nic-1", Namespace: ns, Generation: 1},
		Spec: ionosv1alpha1.NicSpec{
			ForProvider: ionosv1alpha1.NicParameters_2{
				DatacenterIDRef: &v1.NamespacedReference{Name: "workspace-1", Namespace: ns},
				ServerIDRef:     &v1.NamespacedReference{Name: "instance-1", Namespace: ns},
				LanRef:          &v1.NamespacedReference{Name: "lan-1", Namespace: ns},
				DHCP:            new(true),
				FirewallActive:  new(false),
				Ips:             []*string{new("31.70.129.98")},
			},
		},
	}
	nic.SetConditions(v1.Available().WithObservedGeneration(1))
	c := fakeclient.NewClientBuilder().WithScheme(instanceScheme(t)).WithObjects(nic).Build()
	store := NewInstanceStore(c, testLogger())

	// publicIP="" mirrors readNicNetworking's DHCP-fallback result (no PublicIpRefs).
	if err := store.ensureNic(context.Background(), testInstance(), ns, "nic-1", "lan-1", ""); err != nil {
		t.Fatalf("ensureNic = %v, want nil (Ready, DHCP fallback accepts whatever ips is)", err)
	}

	got := &ionosv1alpha1.Nic{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "nic-1"}, got); err != nil {
		t.Fatal(err)
	}
	if len(got.Spec.ForProvider.Ips) != 1 || got.Spec.ForProvider.Ips[0] == nil || *got.Spec.ForProvider.Ips[0] != "31.70.129.98" {
		t.Fatalf("nic ips = %v, want unchanged [31.70.129.98] (DHCP fallback must not touch it)", got.Spec.ForProvider.Ips)
	}
}

// TestPowerOnReassertsStaleNicLan verifies that a restart (PowerOn -> ensureNic) corrects a NIC
// still pointing at a LAN from before the instance's subnet reference changed, instead of only
// ever touching Ips. Without this, a NIC created on lan-old would keep pointing at it forever
// across stop/start even though the domain now resolves to lan-new.
func TestPowerOnReassertsStaleNicLan(t *testing.T) {
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: "tenant-1"})

	nic := &ionosv1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{Name: "nic-1", Namespace: ns, Generation: 1},
		Spec: ionosv1alpha1.NicSpec{
			ForProvider: ionosv1alpha1.NicParameters_2{
				DatacenterIDRef: &v1.NamespacedReference{Name: "workspace-1", Namespace: ns},
				ServerIDRef:     &v1.NamespacedReference{Name: "instance-1", Namespace: ns},
				LanRef:          &v1.NamespacedReference{Name: "lan-old", Namespace: ns},
				DHCP:            new(true),
				FirewallActive:  new(false),
			},
		},
	}
	nic.SetConditions(v1.Available().WithObservedGeneration(1))
	c := fakeclient.NewClientBuilder().WithScheme(instanceScheme(t)).WithObjects(nic).Build()
	store := NewInstanceStore(c, testLogger())

	err := store.ensureNic(context.Background(), testInstance(), ns, "nic-1", "lan-new", "")
	if !errors.Is(err, backend.ErrStillProcessing) {
		t.Fatalf("ensureNic = %v, want ErrStillProcessing (spec just updated, not yet reconciled)", err)
	}

	got := &ionosv1alpha1.Nic{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "nic-1"}, got); err != nil {
		t.Fatal(err)
	}
	if got.Spec.ForProvider.LanRef == nil || got.Spec.ForProvider.LanRef.Name != "lan-new" {
		t.Fatalf("nic lanRef = %v, want lan-new", got.Spec.ForProvider.LanRef)
	}
}

// TestDeleteTearsDownNicVolumeThenServer verifies ordering: while the Nic
// still has a pending finalizer (deletion in progress, mirroring a real
// provider), Delete must return ErrStillProcessing and must NOT touch the
// boot Volume or Server yet.
func TestDeleteTearsDownNicVolumeThenServer(t *testing.T) {
	ns := instanceNamespace(testInstance())

	nic := &ionosv1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{Name: "nic-1", Namespace: ns, Finalizers: []string{"test.finalizer/keep"}},
	}
	vol := &ionosv1alpha1.Volume{ObjectMeta: metav1.ObjectMeta{Name: "block-storage-1", Namespace: ns}}
	srv := &ionosv1alpha1.Server{ObjectMeta: metav1.ObjectMeta{Name: "instance-1", Namespace: ns}}
	c := fakeclient.NewClientBuilder().WithScheme(instanceScheme(t)).WithObjects(nic, vol, srv).Build()
	store := NewInstanceStore(c, testLogger())

	if err := store.Delete(context.Background(), testInstance()); !errors.Is(err, backend.ErrStillProcessing) {
		t.Fatalf("Delete = %v, want ErrStillProcessing (nic deletion in progress)", err)
	}

	gotNic := &ionosv1alpha1.Nic{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "nic-1"}, gotNic); err != nil {
		t.Fatalf("nic should still be present (pending finalizer): %v", err)
	}
	if gotNic.GetDeletionTimestamp().IsZero() {
		t.Fatal("nic deletion was not requested")
	}

	// Volume and Server must be untouched since Nic teardown returned early.
	gotVol := &ionosv1alpha1.Volume{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "block-storage-1"}, gotVol); err != nil {
		t.Fatalf("volume should still exist while nic teardown pending: %v", err)
	}
	gotSrv := &ionosv1alpha1.Server{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "instance-1"}, gotSrv); err != nil {
		t.Fatalf("server should still exist while nic teardown pending: %v", err)
	}
}
