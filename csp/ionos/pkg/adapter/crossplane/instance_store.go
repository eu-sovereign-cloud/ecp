package crossplane

import (
	"context"
	"encoding/base64"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	v2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// IONOS Server VMState values (enterprise servers support RUNNING/SHUTOFF; SUSPENDED is
// cube-only and unused here).
const (
	serverVMStateRunning = "RUNNING"
	serverVMStateShutoff = "SHUTOFF"
)

var _ port.InstanceStore = (*InstanceStore)(nil)

type InstanceStore struct {
	base
}

func NewInstanceStore(c client.Client, logger *slog.Logger) *InstanceStore {
	return &InstanceStore{base{client: c, logger: logger}}
}

// instanceNamespace is where the Server and its instance-owned boot Volume/primary Nic live.
// Instance names (and BlockStorage/NIC names) are only workspace-unique, not tenant-unique, so
// this must include the workspace: two workspaces in the same tenant can both name an instance
// "web", and without the workspace in the namespace hash the second PowerOn would find and take
// over the first workspace's Server (wrong datacenter and all).
func instanceNamespace(domain *instancedom.Instance) string {
	return k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant(), Workspace: domain.GetWorkspace()})
}

// datacenterNamespace is the tenant-wide namespace the Workspace's Datacenter (and Lan) CRs live
// in — shared across all of a tenant's workspaces, since a Datacenter's own name (the workspace
// name) is already tenant-unique.
func datacenterNamespace(domain *instancedom.Instance) string {
	return k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
}

// Create is a no-op: real provisioning happens on PowerOn, when the full
// Instance context (image, SSH keys, user-data, networking) is available.
func (a *InstanceStore) Create(ctx context.Context, domain *instancedom.Instance) error {
	a.logger.Info("instance create: no-op, provisioning deferred to power-on", "name", domain.GetName())
	return nil
}

// Delete tears down NIC -> boot Volume -> Server, in order. The reserved IPBlock
// is owned by the PublicIP plugin and is not deleted here.
func (a *InstanceStore) Delete(ctx context.Context, domain *instancedom.Instance) error {
	ns := instanceNamespace(domain)

	if domain.Spec.PrimaryNicRef != nil {
		nicName := commonbackend.ParseReference(*domain.Spec.PrimaryNicRef, "").Name
		nic := &ionosv1alpha1.Nic{
			TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Nic_Kind},
			ObjectMeta: metav1.ObjectMeta{Name: nicName, Namespace: ns},
		}
		if err := a.deleteCR(ctx, nic); err != nil {
			return err
		}
	}

	bootVolName := commonbackend.ParseReference(domain.Spec.BootVolume.DeviceRef, "").Name
	vol := &ionosv1alpha1.Volume{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Volume_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: bootVolName, Namespace: ns},
	}
	if err := a.deleteCR(ctx, vol); err != nil {
		return err
	}

	srv := &ionosv1alpha1.Server{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Server_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: ns},
	}
	return a.deleteCR(ctx, srv)
}

// PowerOn provisions the Server SHUTOFF, attaches the boot volume and primary NIC
// while it's off, and only flips it to RUNNING once both are ready. A Server booted
// RUNNING before its boot device is attached will not automatically boot from a
// volume hot-attached afterwards, so the device must be in place before the first
// boot.
func (a *InstanceStore) PowerOn(ctx context.Context, domain *instancedom.Instance) error {
	ns := instanceNamespace(domain)

	// 1. Workspace datacenter must be ready.
	dc := &ionosv1alpha1.Datacenter{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Datacenter_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetWorkspace(), Namespace: datacenterNamespace(domain)},
	}
	if err := a.checkExisting(ctx, dc); err != nil {
		return err
	}

	// 2. SKU -> cores/RAM.
	cores, ramMB, err := a.readSKU(ctx, domain.Spec.SkuRef, domain.GetTenant())
	if err != nil {
		return err
	}

	// 3. Server (ENTERPRISE, SHUTOFF). Idempotent; nil only when ready.
	if err := a.ensureServer(ctx, domain, ns, cores, ramMB); err != nil {
		return err
	}

	// 4. Boot volume attached to the server (image + SSH keys + user-data).
	bootVolName := commonbackend.ParseReference(domain.Spec.BootVolume.DeviceRef, domain.GetTenant()).Name
	alias, sizeGB, err := a.readBootImageAlias(ctx, domain.Spec.BootVolume.DeviceRef, domain.GetTenant(), domain.GetWorkspace())
	if err != nil {
		return err
	}
	if err := a.createCR(ctx, a.newBootVolume(domain, ns, bootVolName, alias, sizeGB)); err != nil {
		return err
	}

	// 5. Primary NIC on the public LAN with the reserved public IP.
	if domain.Spec.PrimaryNicRef != nil {
		nicName := commonbackend.ParseReference(*domain.Spec.PrimaryNicRef, domain.GetTenant()).Name
		net, err := a.readNicNetworking(ctx, *domain.Spec.PrimaryNicRef, domain.GetTenant(), domain.GetWorkspace())
		if err != nil {
			return err
		}
		if err := a.ensureNic(ctx, domain, ns, nicName, net); err != nil {
			return err
		}
	}

	// 6. Boot device (and NIC, if any) are attached and ready: flip the Server to
	// RUNNING as the final step.
	return a.ensureServerRunning(ctx, domain, ns)
}

// PowerOff shuts down the Server CR. Idempotent; nil only once the provider
// reports the shutdown reconciled.
func (a *InstanceStore) PowerOff(ctx context.Context, domain *instancedom.Instance) error {
	ns := instanceNamespace(domain)
	srv := &ionosv1alpha1.Server{}
	if err := a.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: domain.GetName()}, srv); err != nil {
		a.logger.Error("failed to get server", "name", domain.GetName(), "error", err)
		return err
	}

	if srv.Spec.ForProvider.VMState != nil && *srv.Spec.ForProvider.VMState == serverVMStateShutoff {
		return a.checkExisting(ctx, srv)
	}
	srv.Spec.ForProvider.VMState = new(serverVMStateShutoff)
	return a.updateCR(ctx, srv)
}

// ensureServer creates the Server (SHUTOFF) if it doesn't exist yet, or just waits for it to
// be Ready otherwise — regardless of its current VMState, since a restart's Server may already
// be Ready-but-SHUTOFF (left that way by a prior PowerOff) and that's fine at this stage: the
// boot volume and NIC only need the Server to exist, not to be RUNNING. Flipping to RUNNING is
// ensureServerRunning's job, and only once those are attached.
func (a *InstanceStore) ensureServer(ctx context.Context, domain *instancedom.Instance, ns string, cores, ramMB float64) error {
	srv := &ionosv1alpha1.Server{}
	err := a.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: domain.GetName()}, srv)
	switch {
	case apierrors.IsNotFound(err):
		return a.createCR(ctx, a.newServer(domain, ns, cores, ramMB))
	case err != nil:
		a.logger.Error("failed to get server", "name", domain.GetName(), "error", err)
		return err
	default:
		return a.checkExisting(ctx, srv)
	}
}

// ensureServerRunning flips an existing, Ready Server to RUNNING — the final PowerOn step, run
// only once the boot volume and NIC are attached. createCR's AlreadyExists fallback
// (checkExisting) only checks the Ready condition — it would otherwise treat an already-existing,
// still-Ready-but-SHUTOFF Server as "nothing to do", permanently stalling a restart even though
// the domain layer already recorded the instance as powered on.
func (a *InstanceStore) ensureServerRunning(ctx context.Context, domain *instancedom.Instance, ns string) error {
	srv := &ionosv1alpha1.Server{}
	if err := a.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: domain.GetName()}, srv); err != nil {
		a.logger.Error("failed to get server", "name", domain.GetName(), "error", err)
		return err
	}
	if srv.Spec.ForProvider.VMState != nil && *srv.Spec.ForProvider.VMState == serverVMStateRunning {
		return a.checkExisting(ctx, srv)
	}
	srv.Spec.ForProvider.VMState = new(serverVMStateRunning)
	return a.updateCR(ctx, srv)
}

// newServer builds the Server SHUTOFF: it must exist so the boot Volume and NIC can
// reference it, but must not boot until its boot device is attached. ensureServerRunning
// flips it to RUNNING once that's done.
func (a *InstanceStore) newServer(domain *instancedom.Instance, ns string, cores, ramMB float64) *ionosv1alpha1.Server {
	sshKeys := toPtrSlice(domain.Spec.SshKeys)
	return &ionosv1alpha1.Server{
		TypeMeta:   metav1.TypeMeta{APIVersion: ionosv1alpha1.CRDGroupVersion.String(), Kind: ionosv1alpha1.Server_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: ns},
		Spec: ionosv1alpha1.ServerSpec{
			ForProvider: ionosv1alpha1.ServerParameters{
				Name:             new(domain.GetName()),
				Type:             new("ENTERPRISE"),
				Cores:            new(cores),
				RAM:              new(ramMB),
				AvailabilityZone: new(translateZone(domain.Spec.Zone)),
				VMState:          new(serverVMStateShutoff),
				DatacenterIDRef:  &v1.NamespacedReference{Name: domain.GetWorkspace(), Namespace: datacenterNamespace(domain)},
				SSHKeys:          sshKeys,
			},
			ManagedResourceSpec: providerConfig(),
		},
	}
}

func (a *InstanceStore) newBootVolume(domain *instancedom.Instance, ns, name, alias string, sizeGB int) *ionosv1alpha1.Volume {
	vol := &ionosv1alpha1.Volume{
		TypeMeta:   metav1.TypeMeta{APIVersion: ionosv1alpha1.CRDGroupVersion.String(), Kind: ionosv1alpha1.Volume_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: ionosv1alpha1.VolumeSpec{
			ForProvider: ionosv1alpha1.VolumeParameters_2{
				DatacenterIDRef:  &v1.NamespacedReference{Name: domain.GetWorkspace(), Namespace: datacenterNamespace(domain)},
				ServerIDRef:      &v1.NamespacedReference{Name: domain.GetName(), Namespace: ns},
				Name:             new(name),
				Size:             new(float64(sizeGB)),
				DiskType:         new("SSD"),
				AvailabilityZone: new("AUTO"),
				ImageName:        new(alias),
				SSHKeys:          toPtrSlice(domain.Spec.SshKeys),
			},
			ManagedResourceSpec: providerConfig(),
		},
	}
	if domain.Spec.UserData != "" {
		vol.Spec.ForProvider.UserData = new(base64.StdEncoding.EncodeToString([]byte(domain.Spec.UserData)))
	}
	return vol
}

// ensureNic creates the primary NIC if it doesn't exist, or re-asserts the current desired spec
// (LAN, server/datacenter refs, DHCP, firewall) on an existing one — otherwise a NIC created
// before a subnet/LAN change would keep pointing at the old LAN across a stop/start. Ips is
// handled separately: it's only corrected when the domain wants a specific reserved public IP
// that isn't (yet) reflected there. When publicIP is empty (no reserved IP — DHCP fallback), ips
// is left untouched no matter its current value: the upjet IONOS provider late-inits whatever
// address IONOS currently reports into spec on its own reconcile schedule, entirely outside our
// control, so trying to force it back to nil here would race that process and never converge —
// permanently blocking PowerOn (and the instance's power state) instead of just letting DHCP do
// its job.
func (a *InstanceStore) ensureNic(ctx context.Context, domain *instancedom.Instance, ns, name string, net nicNetworking) error {
	nic := &ionosv1alpha1.Nic{}
	err := a.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, nic)
	switch {
	case apierrors.IsNotFound(err):
		return a.createCR(ctx, a.newNic(domain, ns, name, net))
	case err != nil:
		a.logger.Error("failed to get nic", "name", name, "error", err)
		return err
	}

	desired := a.newNic(domain, ns, name, net)
	changed := !ptr.Equal(nic.Spec.ForProvider.LanRef, desired.Spec.ForProvider.LanRef) ||
		!ptr.Equal(nic.Spec.ForProvider.ServerIDRef, desired.Spec.ForProvider.ServerIDRef) ||
		!ptr.Equal(nic.Spec.ForProvider.DatacenterIDRef, desired.Spec.ForProvider.DatacenterIDRef) ||
		!ptr.Equal(nic.Spec.ForProvider.DHCP, desired.Spec.ForProvider.DHCP) ||
		!ptr.Equal(nic.Spec.ForProvider.FirewallActive, desired.Spec.ForProvider.FirewallActive) ||
		!sameNamespacedRefs(nic.Spec.ForProvider.SecurityGroupsIdsRefs, desired.Spec.ForProvider.SecurityGroupsIdsRefs)
	if changed {
		nic.Spec.ForProvider.LanRef = desired.Spec.ForProvider.LanRef
		nic.Spec.ForProvider.ServerIDRef = desired.Spec.ForProvider.ServerIDRef
		nic.Spec.ForProvider.DatacenterIDRef = desired.Spec.ForProvider.DatacenterIDRef
		nic.Spec.ForProvider.DHCP = desired.Spec.ForProvider.DHCP
		nic.Spec.ForProvider.FirewallActive = desired.Spec.ForProvider.FirewallActive
		nic.Spec.ForProvider.SecurityGroupsIdsRefs = desired.Spec.ForProvider.SecurityGroupsIdsRefs
	}
	if net.PublicIP != "" && !nicHasReservedIP(nic.Spec.ForProvider.Ips, net.PublicIP) {
		nic.Spec.ForProvider.Ips = []*string{new(net.PublicIP)}
		changed = true
	}
	if changed {
		return a.updateCR(ctx, nic)
	}
	return a.checkExisting(ctx, nic)
}

// nicHasReservedIP reports whether a Nic's current spec.forProvider.ips already reflects the
// single reserved address wantIP.
func nicHasReservedIP(current []*string, wantIP string) bool {
	return len(current) == 1 && current[0] != nil && *current[0] == wantIP
}

func (a *InstanceStore) newNic(domain *instancedom.Instance, ns, name string, net nicNetworking) *ionosv1alpha1.Nic {
	nic := &ionosv1alpha1.Nic{
		TypeMeta:   metav1.TypeMeta{APIVersion: ionosv1alpha1.CRDGroupVersion.String(), Kind: ionosv1alpha1.Nic_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: ionosv1alpha1.NicSpec{
			ForProvider: ionosv1alpha1.NicParameters_2{
				Name:            new(name),
				DatacenterIDRef: &v1.NamespacedReference{Name: domain.GetWorkspace(), Namespace: datacenterNamespace(domain)},
				ServerIDRef:     &v1.NamespacedReference{Name: domain.GetName(), Namespace: ns},
				LanRef:          &v1.NamespacedReference{Name: net.LanName, Namespace: datacenterNamespace(domain)},
				DHCP:            new(true),
				FirewallActive:  new(false),
			},
			ManagedResourceSpec: providerConfig(),
		},
	}
	// Only pin an explicit public IP when one was reserved. When empty, leave Ips
	// unset so IONOS DHCP auto-assigns a public IPv4 on the public LAN.
	if net.PublicIP != "" {
		nic.Spec.ForProvider.Ips = []*string{new(net.PublicIP)}
	}
	nic.Spec.ForProvider.SecurityGroupsIdsRefs = securityGroupRefs(domain, net.SecurityGroupRefs)
	return nic
}

func providerConfig() v2.ManagedResourceSpec {
	return v2.ManagedResourceSpec{
		ProviderConfigReference: &v1.ProviderConfigReference{Name: ProviderConfigName, Kind: ProviderConfigType},
	}
}

func toPtrSlice(in []string) []*string {
	if len(in) == 0 {
		return nil
	}
	out := make([]*string, len(in))
	for i := range in {
		out[i] = new(in[i])
	}
	return out
}

// securityGroupRefs resolves the NSG CRs the IONOS Nic must attach: the security groups the
// SECA NIC names, plus the instance's own if it has one.
//
// Both are folded onto the NIC rather than the instance's going on the Server. IONOS supports
// either level and combines them, so this is a simplicity call, not a semantic one: ensureNic
// already re-asserts the NIC's desired spec on every power-on, while ensureServer deliberately
// does not, and one code path is one place for a group to be dropped or duplicated.
//
// Order follows the spec and duplicates are collapsed, so an unchanged spec always yields an
// identical list — a list that reordered itself between reconciles would read as drift and be
// rewritten forever.
//
// The refs, not the resolved SecurityGroupsIds, are what this plugin owns: Crossplane resolves
// each ref to an NSG's provider ID and writes it into SecurityGroupsIds, so comparing or setting
// that field here would fight the resolver.
func securityGroupRefs(domain *instancedom.Instance, nicRefs []commondomain.Reference) []v1.NamespacedReference {
	refs := make([]commondomain.Reference, 0, len(nicRefs)+1)
	refs = append(refs, nicRefs...)
	if domain.Spec.SecurityGroupRef != nil {
		refs = append(refs, *domain.Spec.SecurityGroupRef)
	}
	if len(refs) == 0 {
		return nil
	}

	seen := make(map[v1.NamespacedReference]struct{}, len(refs))
	out := make([]v1.NamespacedReference, 0, len(refs))
	for _, ref := range refs {
		target := commonbackend.ParseReference(ref, domain.GetTenant())
		if target.Workspace == "" {
			target.Workspace = domain.GetWorkspace()
		}
		resolved := v1.NamespacedReference{
			Name: target.Name,
			Namespace: k8sadapter.ComputeNamespace(&resource.Scope{
				Tenant:    target.Tenant,
				Workspace: target.Workspace,
			}),
		}
		if _, dup := seen[resolved]; dup {
			continue
		}
		seen[resolved] = struct{}{}
		out = append(out, resolved)
	}
	return out
}

// sameNamespacedRefs compares two reference lists by what they point at. NamespacedReference
// carries a Policy pointer that this plugin never sets, so == on the struct would compare
// pointer identity rather than meaning.
func sameNamespacedRefs(a, b []v1.NamespacedReference) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name || a[i].Namespace != b[i].Namespace {
			return false
		}
	}
	return true
}
