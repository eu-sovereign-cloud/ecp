package crossplane

import (
	"context"
	"crypto/rand"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	v2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

var _ port.InstanceStore = (*InstanceStore)(nil)

type InstanceStore struct {
	base
}

func NewInstanceStore(c client.Client, logger *slog.Logger) *InstanceStore {
	return &InstanceStore{base{client: c, logger: logger}}
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
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})

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

func (a *InstanceStore) PowerOn(ctx context.Context, domain *instancedom.Instance) error {
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})

	// 1. Workspace datacenter must be ready.
	dc := &ionosv1alpha1.Datacenter{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Datacenter_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetWorkspace(), Namespace: ns},
	}
	if err := a.checkExisting(ctx, dc); err != nil {
		return err
	}

	// 2. SKU -> cores/RAM.
	cores, ramMB, err := a.readSKU(ctx, domain.Spec.SkuRef, domain.GetTenant())
	if err != nil {
		return err
	}

	// 3. Server (ENTERPRISE, RUNNING). Idempotent; nil only when ready.
	if err := a.createCR(ctx, a.newServer(domain, ns, cores, ramMB)); err != nil {
		return err
	}

	// 4. Boot volume attached to the server (image + SSH keys + user-data).
	bootVolName := commonbackend.ParseReference(domain.Spec.BootVolume.DeviceRef, domain.GetTenant()).Name
	alias, sizeGB, err := a.readBootImageAlias(ctx, domain.Spec.BootVolume.DeviceRef, domain.GetTenant())
	if err != nil {
		return err
	}
	if err := a.createCR(ctx, a.newBootVolume(domain, ns, bootVolName, alias, sizeGB)); err != nil {
		return err
	}

	// 5. Primary NIC on the public LAN with the reserved public IP.
	if domain.Spec.PrimaryNicRef == nil {
		return nil
	}
	nicName := commonbackend.ParseReference(*domain.Spec.PrimaryNicRef, domain.GetTenant()).Name
	lanName, publicIP, err := a.readNicNetworking(ctx, *domain.Spec.PrimaryNicRef, domain.GetTenant())
	if err != nil {
		return err
	}
	return a.createCR(ctx, a.newNic(domain, ns, nicName, lanName, publicIP))
}

// PowerOff shuts down the Server CR. Idempotent; nil only once the provider
// reports the shutdown reconciled.
func (a *InstanceStore) PowerOff(ctx context.Context, domain *instancedom.Instance) error {
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
	srv := &ionosv1alpha1.Server{}
	if err := a.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: domain.GetName()}, srv); err != nil {
		a.logger.Error("failed to get server", "name", domain.GetName(), "error", err)
		return err
	}
	if srv.Spec.ForProvider.VMState != nil && *srv.Spec.ForProvider.VMState == "SHUTOFF" {
		return a.checkExisting(ctx, srv)
	}
	srv.Spec.ForProvider.VMState = new("SHUTOFF")
	return a.updateCR(ctx, srv)
}

func (a *InstanceStore) newServer(domain *instancedom.Instance, ns string, cores, ramMB float64) *ionosv1alpha1.Server {
	sshKeys := toPtrSlice(domain.Spec.SshKeys)
	return &ionosv1alpha1.Server{
		TypeMeta:   metav1.TypeMeta{APIVersion: ionosv1alpha1.CRDGroupVersion.String(), Kind: ionosv1alpha1.Server_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: ns},
		Spec: ionosv1alpha1.ServerSpec{
			ForProvider: ionosv1alpha1.ServerParameters{
				Type:             new("ENTERPRISE"),
				Cores:            new(cores),
				RAM:              new(ramMB),
				AvailabilityZone: new(translateZone(domain.Spec.Zone)),
				VMState:          new("RUNNING"),
				DatacenterIDRef:  &v1.NamespacedReference{Name: domain.GetWorkspace(), Namespace: ns},
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
				DatacenterIDRef:  &v1.NamespacedReference{Name: domain.GetWorkspace(), Namespace: ns},
				ServerIDRef:      &v1.NamespacedReference{Name: domain.GetName(), Namespace: ns},
				Name:             new(name),
				Size:             new(float64(sizeGB)),
				DiskType:         new("SSD"),
				AvailabilityZone: new("AUTO"),
				ImageName:        new(alias),
				ImagePassword:    new(randomPassword()),
				SSHKeys:          toPtrSlice(domain.Spec.SshKeys),
			},
			ManagedResourceSpec: providerConfig(),
		},
	}
	if domain.Spec.UserData != "" {
		vol.Spec.ForProvider.UserData = new(domain.Spec.UserData)
	}
	return vol
}

func (a *InstanceStore) newNic(domain *instancedom.Instance, ns, name, lanName, publicIP string) *ionosv1alpha1.Nic {
	nic := &ionosv1alpha1.Nic{
		TypeMeta:   metav1.TypeMeta{APIVersion: ionosv1alpha1.CRDGroupVersion.String(), Kind: ionosv1alpha1.Nic_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: ionosv1alpha1.NicSpec{
			ForProvider: ionosv1alpha1.NicParameters_2{
				Name:            new(name),
				DatacenterIDRef: &v1.NamespacedReference{Name: domain.GetWorkspace(), Namespace: ns},
				ServerIDRef:     &v1.NamespacedReference{Name: domain.GetName(), Namespace: ns},
				LanRef:          &v1.NamespacedReference{Name: lanName, Namespace: ns},
				DHCP:            new(true),
				FirewallActive:  new(false),
			},
			ManagedResourceSpec: providerConfig(),
		},
	}
	// Only pin an explicit public IP when one was reserved. When empty, leave Ips
	// unset so IONOS DHCP auto-assigns a public IPv4 on the public LAN.
	if publicIP != "" {
		nic.Spec.ForProvider.Ips = []*string{new(publicIP)}
	}
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

const passwordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomPassword() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = passwordCharset[int(b[i])%len(passwordCharset)]
	}
	return string(b)
}
